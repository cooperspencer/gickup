package types

import (
	"fmt"
	"os"
	"regexp"
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

type PrometheusConfig struct {
	ListenAddr string `yaml:"listen_addr"`
	Endpoint   string `yaml:"endpoint"`
}

type Metrics struct {
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
	missing := false
	for key, value := range theMap {
		if value == "" {
			log.Warn().Str("expectedButMissing", key).Msg(
				"A configuration value is expected but not present. Ensure all required configuration is present.")
			missing = true
		}
	}

	return !missing
}

func (conf Conf) HasAllPrometheusConf() bool {
	if len(conf.Metrics.Prometheus.ListenAddr) == 0 && len(conf.Metrics.Prometheus.Endpoint) == 0 {
		return false
	} else {
		checks := map[string]string{
			"listenaddr": conf.Metrics.Prometheus.ListenAddr,
			"endpoint":   conf.Metrics.Prometheus.Endpoint,
		}

		ok := CheckAllValuesOrNone("prometheus", checks)

		if !ok {
			log.Fatal().Str("monitoring", "prometheus").Msg(
				"Fix the values in the configuration.")
		}

		return ok
	}
}

func (conf Conf) MissingCronSpec() bool {
	return conf.Cron == ""
}

func ParseCronSpec(spec string) (cron.Schedule, error) {
	sched, err := cron.ParseStandard(spec)

	if err != nil {
		log.Error().Str("spec", spec).Msg(err.Error())
	}

	return sched, err
}

func (conf Conf) GetNextRun() (*time.Time, error) {
	if conf.MissingCronSpec() {
		return nil, fmt.Errorf("cron unspecified")
	}
	parsedSched, err := ParseCronSpec(conf.Cron)
	if err != nil {
		return nil, err
	}
	next := parsedSched.Next(time.Now())
	return &next, nil
}

func (conf Conf) HasValidCronSpec() bool {
	if conf.MissingCronSpec() {
		return false
	}

	_, err := ParseCronSpec(conf.Cron)

	return err == nil
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
	TokenFile   string   `yaml:"token_file"`
	User        string   `yaml:"user"`
	SSH         bool     `yaml:"ssh"`
	SSHKey      string   `yaml:"sshkey"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
	Url         string   `yaml:"url"`
	Exclude     []string `yaml:"exclude"`
	ExcludeOrgs []string `yaml:"excludeorgs"`
	Include     []string `yaml:"include"`
	IncludeOrgs []string `yaml:"includeorgs"`
	Wiki        bool     `yaml:"wiki"`
	Starred     bool     `yaml:"starred"`
}

func (grepo GenRepo) GetToken() string {
	token, err := resolveToken(grepo.Token, grepo.TokenFile)

	if err != nil {
		log.Fatal().
			Str("url", grepo.Url).
			Str("tokenfile", grepo.TokenFile).
			Err(err)
	}

	return token
}

func resolveToken(tokenString string, tokenFile string) (string, error) {
	if tokenString != "" {
		return tokenString, nil
	}

	if tokenFile != "" {
		data, err := os.ReadFile(tokenFile)

		if err != nil {
			return "", err
		}

		log.Info().
			Int("bytes", len(data)).
			Str("path", tokenFile).
			Msg("Read token file")

		tokenData := strings.ReplaceAll(string(data), "\n", "")
		return tokenData, nil

	}
	return "", fmt.Errorf("no token or tokenfile was specified in config when one was expected")
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
	Red      = color.FgRed.Render
	Green    = color.FgGreen.Render
	Blue     = color.FgBlue.Render
	DotGitRx = regexp.MustCompile(`\.git$`)
)

func GetMap(excludes []string) map[string]bool {
	excludemap := make(map[string]bool)
	for _, exclude := range excludes {
		excludemap[exclude] = true
	}
	return excludemap
}
