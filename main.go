package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/alecthomas/kong"
	"github.com/gookit/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

func (v versionFlag) BeforeApply() error {
	fmt.Println("v0.9.4")
	os.Exit(0)
	return nil
}

func (v dryrunFlag) BeforeApply() error {
	dry = true
	return nil
}

var cli struct {
	Configfile string `arg required name:"conf" help:"path to the configfile." type:"existingfile"`
	Version    versionFlag
	Dry        dryrunFlag `flag name:"dryrun" help:"make a dry-run."`
}

var (
	red   = color.FgRed.Render
	green = color.FgGreen.Render
	blue  = color.FgBlue.Render
	dry   = false
)

func ReadConfigfile(configfile string) *Conf {
	cfgdata, err := ioutil.ReadFile(configfile)

	if err != nil {
		log.Panic().Str("stage", "readconfig").Str("file", configfile).Msgf("Cannot open config file from %s", red(configfile))
	}

	t := Conf{}

	err = yaml.Unmarshal([]byte(cfgdata), &t)

	if err != nil {
		log.Panic().Str("stage", "readconfig").Str("file", configfile).Msg("Cannot map yml config file to interface, possible syntax error")
	}

	return &t
}

func GetExcludedMap(excludes []string) map[string]bool {
	excludemap := make(map[string]bool)
	for _, exclude := range excludes {
		excludemap[exclude] = true
	}
	return excludemap
}

func Backup(repos []Repo, conf *Conf) {
	for _, r := range repos {
		log.Info().Str("stage", "backup").Msgf("starting backup for %s", r.Url)
		for _, d := range conf.Destination.Local {
			Locally(r, d)
		}
		for _, d := range conf.Destination.Gitea {
			BackupGitea(r, d)
		}
		for _, d := range conf.Destination.Gogs {
			BackupGogs(r, d)
		}
		for _, d := range conf.Destination.Gitlab {
			BackupGitlab(r, d)
		}
	}
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	kong.Parse(&cli, kong.Name("gickup"), kong.Description("a tool to backup all your favorite repos"))

	if dry {
		log.Info().Str("dry", "true").Msgf("this is a %s", blue("dry run"))
	}

	log.Info().Str("file", cli.Configfile).Msgf("Reading %s", green(cli.Configfile))
	conf := ReadConfigfile(cli.Configfile)

	// Github
	repos := getGithub(conf)
	Backup(repos, conf)

	// Gitea
	repos = getGitea(conf)
	Backup(repos, conf)

	// Gogs
	repos = getGogs(conf)
	Backup(repos, conf)

	// Gitlab
	repos = getGitlab(conf)
	Backup(repos, conf)

	//Bitbucket
	repos = getBitbucket(conf)
	Backup(repos, conf)
}
