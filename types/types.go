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

func (dest Destination) Count() int {
	return len(dest.Gogs) +
		len(dest.Gitea) +
		len(dest.Local) +
		len(dest.Github) +
		len(dest.Gitlab)
}

// Local
type Local struct {
	Path       string `yaml:"path"`
	Structured bool   `yaml:"structured"`
}

// Conf
type Conf struct {
	Source      Source      `yaml:"source"`
	Destination Destination `yaml:"destination"`
	Cron        string      `yaml:"cron"`
	Log         Logging     `yaml:"log"`
	Metrics     Metrics     `yaml:"metrics"`
}

type InfluxDb2Config struct {
	Bucket string `yaml:"bucket"`
	Org    string `yaml:"org"`
	Token  string `yaml:"token"`
	Url    string `yaml:"url"`
}

type PrometheusConfig struct {
	ListenAddr string `yaml:"listen_addr"`
	Endpoint   string `yaml:"endpoint"`
}

type Metrics struct {
	InfluxDb2  InfluxDb2Config  `yaml:"influxdb2"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
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

func CheckAllValuesOrNone(parent string, theMap map[string]string) bool {
	allEmpty := true

	for key, value := range theMap {
		thisOneIsEmpty := value == ""
		if !allEmpty && thisOneIsEmpty {
			log.Fatal().Str("expectedButMissing", key).Msg(
				"A configuration value is expected but not present. Ensure all required configuration is present.")
		}
		if !thisOneIsEmpty {
			allEmpty = false
		}
	}

	return true
}

func (conf Conf) HasAllPrometheusConf() bool {
	checks := map[string]string{
		"listenaddr": conf.Metrics.Prometheus.ListenAddr,
		"endpoint":   conf.Metrics.Prometheus.Endpoint,
	}
	return CheckAllValuesOrNone("prometheus", checks)
}

func (conf Conf) HasAllInfluxDB2Conf() bool {
	checks := map[string]string{
		"bucket": conf.Metrics.InfluxDb2.Bucket,
		"org":    conf.Metrics.InfluxDb2.Org,
		"token":  conf.Metrics.InfluxDb2.Token,
		"url":    conf.Metrics.InfluxDb2.Url,
	}
	return CheckAllValuesOrNone("influxdb2", checks)
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

func (source Source) Count() int {
	return len(source.Gogs) +
		len(source.Gitea) +
		len(source.BitBucket) +
		len(source.Github) +
		len(source.Gitlab)
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
	Owner         string
	Hoster        string
}

// Site
type Site struct {
	Url  string
	User string
	Port int
}

func GetHost(url string) string {
	if strings.Contains(url, "http://") {
		url = strings.Split(url, "http://")[1]
	}
	if strings.Contains(url, "https://") {
		url = strings.Split(url, "https://")[1]
	}
	if strings.Contains(url, "/") {
		url = strings.Split(url, "/")[0]
	}
	return url
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
