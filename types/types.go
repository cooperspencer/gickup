package types

import (
	"strconv"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// Destination
type Destination struct {
	Gitlab []GenRepo `yaml:"gitlab"`
	Local  []Local   `yaml:"local"`
	Github []GenRepo `yaml:"github"`
	Gitea  []GenRepo `yaml:"gitea"`
	Gogs   []GenRepo `yaml:"gogs"`
}

// Local
type Local struct {
	Path string `yaml:"path"`
}

// Conf
type Conf struct {
	Source      Source      `yaml:"source"`
	Destination Destination `yaml:"destination"`
	Cron        string      `yaml:"cron"`
	Log         Logging     `yaml:"log"`
}

type Logging struct {
	Timeformat  string      `yaml:"timeformat"`
	FileLogging FileLogging `yaml:"file-logging"`
}

type FileLogging struct {
	Dir    string `yaml:"dir"`
	File   string `yaml:"file"`
	MaxAge int    `yaml:"maxage"`
}

func (conf Conf) MissingCronSpec() bool {
	return conf.Cron == ""
}

func ParseCronSpec(spec string) cron.Schedule {
	sched, err := cron.ParseStandard(spec)

	if err != nil {
		log.Error().Str("spec", spec).Msg(err.Error())
	}

	return sched
}

func (conf Conf) HasValidCronSpec() bool {
	if conf.MissingCronSpec() {
		return false
	}

	parsedSched := ParseCronSpec(conf.Cron)

	if parsedSched != nil {
		nextRun := parsedSched.Next(time.Now()).String()
		log.Info().Str("next", nextRun).Str("cron", conf.Cron).Msg("Next cron run")
	}

	return parsedSched != nil
}

// Source
type Source struct {
	Gogs      []GenRepo `yaml:"gogs"`
	Gitlab    []GenRepo `yaml:"gitlab"`
	Github    []GenRepo `yaml:"github"`
	Gitea     []GenRepo `yaml:"gitea"`
	BitBucket []GenRepo `yaml:"bitbucket"`
}

// Generell Repo
type GenRepo struct {
	Token       string   `yaml:"token"`
	User        string   `yaml:"user"`
	SSH         bool     `yaml:"ssh"`
	SSHKey      string   `yaml:"sshkey"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
	Url         string   `yaml:"url"`
	Exclude     []string `yaml:"exclude"`
	ExcludeOrgs []string `yaml:"excludeorgs"`
	Include     []string `yaml:"include"`
}

// Repo
type Repo struct {
	Name          string
	Url           string
	SshUrl        string
	Token         string
	Defaultbranch string
	Origin        GenRepo
}

// Site
type Site struct {
	Url  string
	User string
	Port int
}

func (s *Site) GetValues(url string) error {
	if strings.HasPrefix(url, "ssh://") {
		url = strings.Split(url, "ssh://")[1]
		userurl := strings.Split(url, "@")
		s.User = userurl[0]
		urlport := strings.Split(userurl[1], ":")
		s.Url = urlport[0]
		portstring := strings.Split(urlport[1], "/")[0]
		port, err := strconv.Atoi(portstring)
		if err != nil {
			return err
		}
		s.Port = port
	} else {
		userurl := strings.Split(url, "@")
		s.User = userurl[0]
		urlport := strings.Split(userurl[1], ":")
		s.Url = urlport[0]
		s.Port = 22
	}
	return nil
}

var (
	Red   = color.FgRed.Render
	Green = color.FgGreen.Render
	Blue  = color.FgBlue.Render
)

func GetMap(excludes []string) map[string]bool {
	excludemap := make(map[string]bool)
	for _, exclude := range excludes {
		excludemap[exclude] = true
	}
	return excludemap
}
