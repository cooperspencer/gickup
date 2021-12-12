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
	"gickup/types"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type versionFlag bool

func (v versionFlag) BeforeApply() error {
	fmt.Println("v0.9.5")
	os.Exit(0)
	return nil
}

type dryrunFlag bool

func (v dryrunFlag) BeforeApply() error {
	dry = true
	return nil
}

var cli struct {
	Configfile string `arg requitypes.Red name:"conf" help:"path to the configfile." type:"existingfile"`
	Version    versionFlag
	Dry        dryrunFlag `flag name:"dryrun" help:"make a dry-run."`
}

var (
	dry = false
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
			local.Locally(r, d, dry)
		}
		for _, d := range conf.Destination.Gitea {
			gitea.Backup(r, d, dry)
		}
		for _, d := range conf.Destination.Gogs {
			gogs.Backup(r, d, dry)
		}
		for _, d := range conf.Destination.Gitlab {
			gitlab.Backup(r, d, dry)
		}
	}
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	kong.Parse(&cli, kong.Name("gickup"), kong.Description("a tool to backup all your favorite repos"))

	if dry {
		log.Info().Str("dry", "true").Msgf("this is a %s", types.Blue("dry run"))
	}

	log.Info().Str("file", cli.Configfile).Msgf("Reading %s", types.Green(cli.Configfile))
	conf := ReadConfigfile(cli.Configfile)

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
}
