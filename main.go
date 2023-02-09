package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cooperspencer/gickup/onedev"
	"github.com/cooperspencer/gickup/sourcehut"

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
	"github.com/cooperspencer/gickup/whatever"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var cli struct {
	Configfiles []string `arg name:"conf" help:"Path to the configfile." default:"conf.yml"`
	Version     bool     `flag name:"version" help:"Show version."`
	Dry         bool     `flag name:"dryrun" help:"Make a dry-run."`
	Quiet       bool     `flag name:"quiet" help:"Output only warnings, errors, and fatal messages to stderr log output"`
	Silent      bool     `flag name:"silent" help:"Suppress all stderr log output"`
}

var version = "unknown"

func readConfigFile(configfile string) []*types.Conf {
	conf := []*types.Conf{}
	cfgdata, err := os.ReadFile(configfile)
	if err != nil {
		log.Fatal().
			Str("stage", "readconfig").
			Str("file", configfile).
			Msgf("Cannot open config file from %s", types.Red(configfile))
	}

	//	t := types.Conf{}

	dec := yaml.NewDecoder(bytes.NewReader(cfgdata))

	//	err = yaml.Unmarshal(cfgdata, &t)

	i := 0
	for {
		var c *types.Conf
		err = dec.Decode(&c)
		if err == io.EOF {
			break
		} else if err != nil {
			if len(conf) > 0 {
				log.Fatal().
					Str("stage", "readconfig").
					Str("file", configfile).
					Msgf("an error occured in the %d place of %s", i, configfile)
			} else {
				log.Fatal().
					Str("stage", "readconfig").
					Str("file", configfile).
					Msg("Cannot map yml config file to interface, possible syntax error")
			}
		}

		if !reflect.ValueOf(c).IsZero() {
			conf = append(conf, c)
			i++
		}
	}

	return conf
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
	// in whatever other strange case
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

			repotime := time.Now()
			status := 0
			if local.Locally(r, d, cli.Dry) {
				prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "local", d.Path).Set(time.Now().Sub(repotime).Seconds())
				status = 1
			}

			prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "local", d.Path).Set(float64(status))
			prometheus.DestinationBackupsComplete.WithLabelValues("local").Inc()
		}

		for _, d := range conf.Destination.Gitea {
			if !strings.HasSuffix(r.Name, ".wiki") {
				repotime := time.Now()
				status := 0
				if gitea.Backup(r, d, cli.Dry) {
					prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitea", d.URL).Set(time.Now().Sub(repotime).Seconds())
					status = 1
				}

				prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitea", d.URL).Set(float64(status))
				prometheus.DestinationBackupsComplete.WithLabelValues("gitea").Inc()
			}
		}

		for _, d := range conf.Destination.Gogs {
			if !strings.HasSuffix(r.Name, ".wiki") {
				repotime := time.Now()
				status := 0
				if gogs.Backup(r, d, cli.Dry) {
					prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "gogs", d.URL).Set(time.Now().Sub(repotime).Seconds())
					status = 1
				}

				prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "gogs", d.URL).Set(float64(status))
				prometheus.DestinationBackupsComplete.WithLabelValues("gogs").Inc()
			}
		}

		for _, d := range conf.Destination.Gitlab {
			if !strings.HasSuffix(r.Name, ".wiki") {
				repotime := time.Now()
				status := 0
				if gitlab.Backup(r, d, cli.Dry) {
					prometheus.RepoTime.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitlab", d.URL).Set(time.Now().Sub(repotime).Seconds())
					status = 1
				}

				prometheus.RepoSuccess.WithLabelValues(r.Hoster, r.Name, r.Owner, "gitlab", d.URL).Set(float64(status))
				prometheus.DestinationBackupsComplete.WithLabelValues("gitlab").Inc()
			}
		}

		prometheus.SourceBackupsComplete.WithLabelValues(r.Name).Inc()
	}
}

func runBackup(conf *types.Conf, num int) {
	log.Info().Msg("Backup run starting")

	numstring := strconv.Itoa(num)

	startTime := time.Now()

	prometheus.JobsStarted.Inc()

	// Github
	repos, ran := github.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("github", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	// Gitea
	repos, ran = gitea.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("gitea", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	// Gogs
	repos, ran = gogs.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("gogs", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	// Gitlab
	repos, ran = gitlab.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("gitlab", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	repos, ran = bitbucket.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("bitbucket", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	repos, ran = whatever.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("whatever", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	repos, ran = onedev.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("onedev", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	repos, ran = sourcehut.Get(conf)
	if ran {
		prometheus.CountReposDiscovered.WithLabelValues("sourcehut", numstring).Set(float64(len(repos)))
	}
	backup(repos, conf)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	prometheus.JobsComplete.Inc()
	prometheus.JobDuration.Observe(duration.Seconds())

	if len(conf.Metrics.Heartbeat.URLs) > 0 {
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

	if len(os.Getenv("GICKUP_TIME_FORMAT")) > 0 {
		timeformat = os.Getenv("GICKUP_TIME_FORMAT")
	}

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

	confs := []*types.Conf{}
	for _, f := range cli.Configfiles {
		log.Info().Str("file", f).
			Msgf("Reading %s", types.Green(f))

		confs = append(confs, readConfigFile(f)...)
	}
	if confs[0].Log.Timeformat == "" {
		confs[0].Log.Timeformat = timeformat
	}

	log.Logger = logger.CreateLogger(confs[0].Log)

	validcron := confs[0].HasValidCronSpec()

	var c *cron.Cron

	if validcron {
		c = cron.New()
		c.Start()
	}

	sourcecount := 0
	destinationcount := 0
	// one pair per source-destination
	for num, conf := range confs {
		pairs := conf.Source.Count() * conf.Destination.Count()
		sourcecount += conf.Source.Count()
		destinationcount += conf.Destination.Count()
		log.Info().
			Int("sources", conf.Source.Count()).
			Int("destinations", conf.Destination.Count()).
			Int("pairs", pairs).
			Msg("Configuration loaded")

		if !conf.HasValidCronSpec() {
			conf.Cron = confs[0].Cron
		}

		if conf.HasValidCronSpec() && validcron {
			conf := conf // https://stackoverflow.com/questions/57095167/how-do-i-create-multiple-cron-function-by-looping-through-a-list
			num := num

			logNextRun(conf)

			_, err := c.AddFunc(conf.Cron, func() {
				runBackup(conf, num)
			})
			if err != nil {
				log.Fatal().
					Int("sources", conf.Source.Count()).
					Int("destinations", conf.Destination.Count()).
					Int("pairs", pairs).
					Msg(err.Error())
			}
		} else {
			runBackup(conf, num)
		}
	}

	if validcron {
		if confs[0].HasAllPrometheusConf() {
			prometheus.CountSourcesConfigured.Add(float64(sourcecount))
			prometheus.CountDestinationsConfigured.Add(float64(destinationcount))
			prometheus.Serve(confs[0].Metrics.Prometheus)
		} else {
			playsForever()
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
