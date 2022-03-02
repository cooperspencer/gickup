package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/cooperspencer/gickup/bitbucket"
	"github.com/cooperspencer/gickup/gitea"
	"github.com/cooperspencer/gickup/github"
	"github.com/cooperspencer/gickup/gitlab"
	"github.com/cooperspencer/gickup/gogs"
	"github.com/cooperspencer/gickup/local"
	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/metrics/heartbeat"
	"github.com/cooperspencer/gickup/metrics/prometheus"
	"github.com/cooperspencer/gickup/types"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var cli struct {
	Configfile string `arg name:"conf" help:"Path to the configfile." default:"conf.yml"`
	Version    bool   `flag name:"version" help:"Show version."`
	Dry        bool   `flag name:"dryrun" help:"Make a dry-run."`
	Quiet      bool   `flag name:"quiet" help:"Output only warnings, errors, and fatal messages to stderr log output"`
	Silent     bool   `flag name:"silent" help:"Suppress all stderr log output"`
}

var version = "unknown"

func readConfigFile(configfile string) *types.Conf {
	cfgdata, err := ioutil.ReadFile(configfile)
	if err != nil {
		log.Fatal().
			Str("stage", "readconfig").
			Str("file", configfile).
			Msgf("Cannot open config file from %s", types.Red(configfile))
	}

	t := types.Conf{}

	err = yaml.Unmarshal(cfgdata, &t)

	if err != nil {
		log.Fatal().
			Str("stage", "readconfig").
			Str("file", configfile).
			Msg("Cannot map yml config file to interface, possible syntax error")
	}

	return &t
}

func getUserHome() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return usr.HomeDir, nil
}

func substituteHomeForTildeInPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	if path == "~" {
		userHome, err := getUserHome()
		if err != nil {
			log.Fatal().
				Str("stage", "local ~ substitution").
				Str("path", path).
				Msg(err.Error())
		} else {
			return userHome
		}
	}

	if strings.HasPrefix(path, "~/") {
		userHome, err := getUserHome()
		if err != nil {
			log.Fatal().
				Str("stage", "local ~/ substitution").
				Str("path", path).
				Msg(err.Error())
		} else {
			return filepath.Join(userHome, path[2:])
		}
	}
	// in any other strange case
	return path
}

func backup(repos []types.Repo, conf *types.Conf) {
	checkedpath := false

	for _, r := range repos {
		log.Info().
			Str("stage", "backup").
			Msgf("starting backup for %s", r.URL)

		for i, d := range conf.Destination.Local {
			if !checkedpath {
				d.Path = substituteHomeForTildeInPath(d.Path)

				path, err := filepath.Abs(d.Path)
				if err != nil {
					log.Fatal().
						Str("stage", "locally").
						Str("path", d.Path).
						Msg(err.Error())
				}

				conf.Destination.Local[i].Path = path
				checkedpath = true
			}

			local.Locally(r, d, cli.Dry)
			prometheus.DestinationBackupsComplete.WithLabelValues("local").Inc()
		}

		for _, d := range conf.Destination.Gitea {
			gitea.Backup(r, d, cli.Dry)
			prometheus.DestinationBackupsComplete.WithLabelValues("gitea").Inc()
		}

		for _, d := range conf.Destination.Gogs {
			gogs.Backup(r, d, cli.Dry)
			prometheus.DestinationBackupsComplete.WithLabelValues("gogs").Inc()
		}

		for _, d := range conf.Destination.Gitlab {
			gitlab.Backup(r, d, cli.Dry)
			prometheus.DestinationBackupsComplete.WithLabelValues("gitlab").Inc()
		}

		prometheus.SourceBackupsComplete.WithLabelValues(r.Name).Inc()
	}
}

func runBackup(conf *types.Conf) {
	log.Info().Msg("Backup run starting")

	startTime := time.Now()

	prometheus.JobsStarted.Inc()

	// Github
	repos := github.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("github").Set(float64(len(repos)))
	backup(repos, conf)

	// Gitea
	repos = gitea.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("gitea").Set(float64(len(repos)))
	backup(repos, conf)

	// Gogs
	repos = gogs.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("gogs").Set(float64(len(repos)))
	backup(repos, conf)

	// Gitlab
	repos = gitlab.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("gitlab").Set(float64(len(repos)))
	backup(repos, conf)

	repos = bitbucket.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("bitbucket").Set(float64(len(repos)))
	backup(repos, conf)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	prometheus.JobsComplete.Inc()
	prometheus.JobDuration.Observe(duration.Seconds())

	if conf.Metrics.Heartbeat.URL != "" {
		heartbeat.Send(conf.Metrics.Heartbeat)
	}

	log.Info().
		Str("duration", duration.String()).
		Msg("Backup run complete")

	if conf.HasValidCronSpec() {
		logNextRun(conf)
	}
}

func playsForever() {
	wait := make(chan struct{})

	for {
		<-wait
	}
}

func main() {
	timeformat := "2006-01-02T15:04:05Z07:00"
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: timeformat,
	})

	kong.Parse(&cli, kong.Name("gickup"),
		kong.Description("a tool to backup all your favorite repos"))

	if cli.Version {
		fmt.Println(version)

		return
	}

	if cli.Quiet {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	if cli.Silent {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	if cli.Dry {
		log.Info().
			Str("dry", "true").
			Msgf("this is a %s", types.Blue("dry run"))
	}

	log.Info().Str("file", cli.Configfile).
		Msgf("Reading %s", types.Green(cli.Configfile))

	conf := readConfigFile(cli.Configfile)
	if conf.Log.Timeformat == "" {
		conf.Log.Timeformat = timeformat
	}

	log.Logger = logger.CreateLogger(conf.Log)

	// one pair per source-destination
	pairs := conf.Source.Count() * conf.Destination.Count()
	log.Info().
		Int("sources", conf.Source.Count()).
		Int("destinations", conf.Destination.Count()).
		Int("pairs", pairs).
		Msg("Configuration loaded")

	if conf.HasValidCronSpec() {
		c := cron.New()

		logNextRun(conf)

		_, err := c.AddFunc(conf.Cron, func() {
			runBackup(conf)
		})
		if err != nil {
			log.Fatal().
				Int("sources", conf.Source.Count()).
				Int("destinations", conf.Destination.Count()).
				Int("pairs", pairs).
				Msg(err.Error())
		}

		c.Start()

		if conf.HasAllPrometheusConf() {
			prometheus.CountSourcesConfigured.Add(float64(conf.Source.Count()))
			prometheus.CountDestinationsConfigured.Add(float64(conf.Destination.Count()))
			prometheus.Serve(conf.Metrics.Prometheus)
		} else {
			playsForever()
		}
	} else {
		runBackup(conf)
	}
}

func logNextRun(conf *types.Conf) {
	nextRun, err := conf.GetNextRun()
	if err == nil {
		log.Info().
			Str("next", nextRun.String()).
			Str("cron", conf.Cron).
			Msg("Next cron run")
	}
}
