package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"gickup/bitbucket"
	"gickup/gitea"
	"gickup/github"
	"gickup/gitlab"
	"gickup/gogs"
	"gickup/local"
	"gickup/logger"
	"gickup/metrics/influx"
	prometheus "gickup/metrics/prometheus"
	"gickup/types"

	"github.com/alecthomas/kong"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

var cli struct {
	Configfile string `arg name:"conf" help:"path to the configfile." default:"conf.yml"`
	Version    bool   `flag name:"version" help:"show version."`
	Dry        bool   `flag name:"dryrun" help:"make a dry-run."`
	Quiet      bool   `flag name:"quiet" help:"turn of commandline output"`
}

var (
	version = "unknown"
)

func ReadConfigfile(configfile string) *types.Conf {
	cfgdata, err := ioutil.ReadFile(configfile)

	if err != nil {
		log.Fatal().Str("stage", "readconfig").Str("file", configfile).Msgf("Cannot open config file from %s", types.Red(configfile))
	}

	t := types.Conf{}

	err = yaml.Unmarshal([]byte(cfgdata), &t)

	if err != nil {
		log.Fatal().Str("stage", "readconfig").Str("file", configfile).Msg("Cannot map yml config file to interface, possible syntax error")
	}

	return &t
}

func GetUserHome() (string, error) {
	usr, err := user.Current()

	if err != nil {
		return "", err
	}

	return usr.HomeDir, nil
}

func SubstituteHomeForTildeInPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	} else {
		if path == "~" {
			userHome, err := GetUserHome()
			if err != nil {
				log.Fatal().Str("stage", "local ~ substitution").Str("path", path).Msg(err.Error())
			} else {
				return userHome
			}
		} else if strings.HasPrefix(path, "~/") {
			userHome, err := GetUserHome()
			if err != nil {
				log.Fatal().Str("stage", "local ~/ substitution").Str("path", path).Msg(err.Error())
			} else {
				return filepath.Join(userHome, path[2:])
			}
		}
	}
	// in any other strange case
	return path
}

func Backup(repos []types.Repo, conf *types.Conf) {
	checkedpath := false
	for _, r := range repos {
		log.Info().Str("stage", "backup").Msgf("starting backup for %s", r.Url)
		for i, d := range conf.Destination.Local {
			if !checkedpath {
				d.Path = SubstituteHomeForTildeInPath(d.Path)
				path, err := filepath.Abs(d.Path)
				if err != nil {
					log.Fatal().Str("stage", "locally").Str("path", d.Path).Msg(err.Error())
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

func RunBackup(conf *types.Conf) {
	log.Info().Msg("Backup run starting")
	startTime := time.Now()

	prometheus.JobsStarted.Inc()

	// Github
	repos := github.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("github").Set(float64(len(repos)))
	Backup(repos, conf)

	// Gitea
	repos = gitea.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("gitea").Set(float64(len(repos)))
	Backup(repos, conf)

	// Gogs
	repos = gogs.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("gogs").Set(float64(len(repos)))
	Backup(repos, conf)

	// Gitlab
	repos = gitlab.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("gitlab").Set(float64(len(repos)))
	Backup(repos, conf)

	//Bitbucket
	repos = bitbucket.Get(conf)
	prometheus.CountReposDiscovered.WithLabelValues("bitbucket").Set(float64(len(repos)))
	Backup(repos, conf)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	prometheus.JobsComplete.Inc()
	prometheus.JobDuration.Observe(duration.Seconds())

	log.Info().
		Float64("duration", duration.Seconds()).
		Msg("Backup run complete")

	if conf.HasValidCronSpec() {
		logNextRun(conf)
	}
}

func PlaysForever() {
	wait := make(chan struct{})
	for {
		<-wait
	}
}

func main() {
	timeformat := "2006-01-02T15:04:05Z07:00"
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: timeformat})

	kong.Parse(&cli, kong.Name("gickup"), kong.Description("a tool to backup all your favorite repos"))

	if cli.Version {
		fmt.Println(version)
	} else {
		if cli.Dry {
			if !cli.Quiet {
				log.Info().Str("dry", "true").Msgf("this is a %s", types.Blue("dry run"))
			}
		}

		if !cli.Quiet {
			log.Info().Str("file", cli.Configfile).Msgf("Reading %s", types.Green(cli.Configfile))
		}
		conf := ReadConfigfile(cli.Configfile)
		if conf.Log.Timeformat == "" {
			conf.Log.Timeformat = timeformat
		}

		log.Logger = logger.CreateLogger(conf.Log, cli.Quiet)

		// one pair per source-destination
		pairs := conf.Source.Count() * conf.Destination.Count()
		log.Info().
			Int("sources", conf.Source.Count()).
			Int("destinations", conf.Destination.Count()).
			Int("pairs", pairs).
			Msg("Configuration loaded")

		if conf.HasAllInfluxDB2Conf() {
			influx.Setup(conf.Metrics.InfluxDb2)
		}

		if conf.HasAllPrometheusConf() {
			prometheus.CountSourcesConfigured.Add(float64(conf.Source.Count()))
			prometheus.CountDestinationsConfigured.Add(float64(conf.Destination.Count()))
		}

		if conf.HasValidCronSpec() {
			c := cron.New()
			logNextRun(conf)

			c.AddFunc(conf.Cron, func() {
				RunBackup(conf)
			})
			c.Start()

			if conf.HasAllPrometheusConf() {
				prometheus.Serve(conf.Metrics.Prometheus)
			} else {
				PlaysForever()
			}
		} else {
			RunBackup(conf)
		}
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
