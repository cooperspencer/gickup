package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gickup/bitbucket"
	"gickup/gitea"
	"gickup/github"
	"gickup/gitlab"
	"gickup/gogs"
	"gickup/local"
	"gickup/logger"
	"gickup/types"

	"github.com/alecthomas/kong"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

var cli struct {
	Configfile string `arg name:"conf" help:"path to the configfile." default:"conf.yml" type:"existingfile"`
	Version    bool   `flag name:"version" help:"show version."`
	Dry        bool   `flag name:"dryrun" help:"make a dry-run."`
}

var (
	version = "v0.9.5"
)

func ReadConfigfile(configfile string) *types.Conf {
	cfgdata, err := ioutil.ReadFile(configfile)

	if err != nil {
		log.Panic().Str("stage", "readconfig").Str("file", configfile).Msgf("Cannot open config file from %s", types.Red(configfile))
	}

	t := types.Conf{}

	err = yaml.Unmarshal([]byte(cfgdata), &t)

	if err != nil {
		log.Panic().Str("stage", "readconfig").Str("file", configfile).Msg("Cannot map yml config file to interface, possible syntax error")
	}

	return &t
}

func Backup(repos []types.Repo, conf *types.Conf) {
	checkedpath := false
	for _, r := range repos {
		log.Info().Str("stage", "backup").Msgf("starting backup for %s", r.Url)
		for i, d := range conf.Destination.Local {
			if !checkedpath {
				path, err := filepath.Abs(d.Path)
				if err != nil {
					log.Panic().Str("stage", "locally").Str("path", d.Path).Msg(err.Error())
				}
				conf.Destination.Local[i].Path = path
				checkedpath = true
			}
			local.Locally(r, d, cli.Dry)
		}
		for _, d := range conf.Destination.Gitea {
			gitea.Backup(r, d, cli.Dry)
		}
		for _, d := range conf.Destination.Gogs {
			gogs.Backup(r, d, cli.Dry)
		}
		for _, d := range conf.Destination.Gitlab {
			gitlab.Backup(r, d, cli.Dry)
		}
	}
}

func RunBackup(conf *types.Conf) {
	// Github
	repos := github.Get(conf)
	Backup(repos, conf)

	// Gitea
	repos = gitea.Get(conf)
	Backup(repos, conf)

	// Gogs
	repos = gogs.Get(conf)
	Backup(repos, conf)

	// Gitlab
	repos = gitlab.Get(conf)
	Backup(repos, conf)

	//Bitbucket
	repos = bitbucket.Get(conf)
	Backup(repos, conf)

	conf.HasValidCronSpec()
}

func PlaysForever() {
	wait := make(chan struct{})
	for {
		<-wait
	}
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02T15:04:05Z07:00"})

	kong.Parse(&cli, kong.Name("gickup"), kong.Description("a tool to backup all your favorite repos"))

	if cli.Version {
		fmt.Println(version)
	} else {
		if cli.Dry {
			log.Info().Str("dry", "true").Msgf("this is a %s", types.Blue("dry run"))
		}

		log.Info().Str("file", cli.Configfile).Msgf("Reading %s", types.Green(cli.Configfile))
		conf := ReadConfigfile(cli.Configfile)

		log.Logger = logger.CreateLogger(conf.Log)

		if conf.HasValidCronSpec() {
			c := cron.New()
			c.AddFunc(conf.Cron, func() { RunBackup(conf) })
			c.Start()
			PlaysForever()
		} else {
			RunBackup(conf)
		}
	}
}
