package types

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/gookit/color"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// Destination TODO.
type Destination struct {
	Gitlab []GenRepo `yaml:"gitlab"`
	Local  []Local   `yaml:"local"`
	Github []GenRepo `yaml:"github"`
	Gitea  []GenRepo `yaml:"gitea"`
	Gogs   []GenRepo `yaml:"gogs"`
}

// Count TODO.
func (dest Destination) Count() int {
	return len(dest.Gogs) +
		len(dest.Gitea) +
		len(dest.Local) +
		len(dest.Github) +
		len(dest.Gitlab)
}

// Local TODO.
type Local struct {
	Bare       bool   `yaml:"bare"`
	Path       string `yaml:"path"`
	Structured bool   `yaml:"structured"`
	Zip        bool   `yaml:"zip"`
	Keep       int    `yaml:"keep"`
}

// Conf TODO.
type Conf struct {
	Source      Source      `yaml:"source"`
	Destination Destination `yaml:"destination"`
	Cron        string      `yaml:"cron"`
	Log         Logging     `yaml:"log"`
	Metrics     Metrics     `yaml:"metrics"`
}

// PrometheusConfig TODO.
type PrometheusConfig struct {
	ListenAddr string `yaml:"listen_addr"`
	Endpoint   string `yaml:"endpoint"`
}

type HeartbeatConfig struct {
	URLs []string `yaml:"urls"`
}

// Metrics TODO.
type Metrics struct {
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Heartbeat  HeartbeatConfig  `yaml:"heartbeat"`
}

// Logging TODO.
type Logging struct {
	Timeformat  string      `yaml:"timeformat"`
	FileLogging FileLogging `yaml:"file-logging"`
}

// FileLogging TODO.
type FileLogging struct {
	Dir    string `yaml:"dir"`
	File   string `yaml:"file"`
	MaxAge int    `yaml:"maxage"`
}

// CheckAllValuesOrNone TODO.
func CheckAllValuesOrNone(parent string, theMap map[string]string) bool {
	var missing bool

	for key, value := range theMap {
		if value == "" {
			log.Warn().Str("expectedButMissing", key).
				Msg("A configuration value is expected but not present. Ensure all required configuration is present.")

			missing = true
		}
	}

	return !missing
}

// HasAllPrometheusConf TODO.
func (conf Conf) HasAllPrometheusConf() bool {
	if len(conf.Metrics.Prometheus.ListenAddr) == 0 && len(conf.Metrics.Prometheus.Endpoint) == 0 {
		return false
	}

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

// MissingCronSpec TODO.
func (conf Conf) MissingCronSpec() bool {
	return conf.Cron == ""
}

// ParseCronSpec TODO.
func ParseCronSpec(spec string) (cron.Schedule, error) {
	sched, err := cron.ParseStandard(spec)
	if err != nil {
		log.Error().Str("spec", spec).Msg(err.Error())
	}

	return sched, err
}

// GetNextRun TODO.
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

// HasValidCronSpec TODO.
func (conf Conf) HasValidCronSpec() bool {
	if conf.MissingCronSpec() {
		return false
	}

	_, err := ParseCronSpec(conf.Cron)

	return err == nil
}

// Source TODO.
type Source struct {
	Gogs      []GenRepo `yaml:"gogs"`
	Gitlab    []GenRepo `yaml:"gitlab"`
	Github    []GenRepo `yaml:"github"`
	Gitea     []GenRepo `yaml:"gitea"`
	BitBucket []GenRepo `yaml:"bitbucket"`
	OneDev    []GenRepo `yaml:"onedev"`
	Sourcehut []GenRepo `yaml:"sourcehut"`
	Any       []GenRepo `yaml:"any"`
}

// Count TODO.
func (source Source) Count() int {
	return len(source.Gogs) +
		len(source.Gitea) +
		len(source.BitBucket) +
		len(source.Github) +
		len(source.Gitlab) +
		len(source.OneDev) +
		len(source.Sourcehut) +
		len(source.Any)
}

// GenRepo Generell Repo.
type GenRepo struct {
	Token       string     `yaml:"token"`
	TokenFile   string     `yaml:"token_file"`
	User        string     `yaml:"user"`
	SSH         bool       `yaml:"ssh"`
	SSHKey      string     `yaml:"sshkey"`
	Username    string     `yaml:"username"`
	Password    string     `yaml:"password"`
	URL         string     `yaml:"url"`
	Exclude     []string   `yaml:"exclude"`
	ExcludeOrgs []string   `yaml:"excludeorgs"`
	Include     []string   `yaml:"include"`
	IncludeOrgs []string   `yaml:"includeorgs"`
	Wiki        bool       `yaml:"wiki"`
	Starred     bool       `yaml:"starred"`
	CreateOrg   bool       `yaml:"createorg"`
	Visibility  Visibility `yaml:"visibility"`
	Filter      Filter     `yaml:"filter"`
	Contributed bool       `yaml:"contributed"`
}

// Visibility struct
type Visibility struct {
	Repositories  string `yaml:"repositories"`
	Organizations string `yaml:"organizations"`
}

// Filter struct
type Filter struct {
	LastActivityString   string `yaml:"lastactivity"`
	LastActivityDuration time.Duration
	Stars                int      `yaml:"stars"`
	Languages            []string `yaml:"languages"`
	ExcludeArchived      bool     `yaml:"excludearchived"`
}

// GetToken TODO.
func (grepo GenRepo) GetToken() string {
	token, err := resolveToken(grepo.Token, grepo.TokenFile)
	if err != nil {
		log.Fatal().
			Str("url", grepo.URL).
			Str("tokenfile", grepo.TokenFile).
			Msg(err.Error())
	}

	return token
}

func (f *Filter) ParseDuration() error {
	rest := strings.Trim(f.LastActivityString, " ")
	date := time.Now()
	parsed := false
	if strings.Contains(rest, "y") {
		durs := strings.Split(rest, "y")
		yearsstring := durs[0]
		if len(durs) >= 2 {
			rest = strings.Join(durs[1:], "")
		}
		years, err := strconv.Atoi(yearsstring)
		if err != nil {
			return err
		}
		date = date.AddDate(years*(-1), 0, 0)
		parsed = true
	}
	if strings.Contains(rest, "M") {
		durs := strings.Split(rest, "M")
		monthsstring := durs[0]
		if len(durs) >= 2 {
			rest = strings.Join(durs[1:], "")
		}
		months, err := strconv.Atoi(monthsstring)
		if err != nil {
			return err
		}
		date = date.AddDate(0, months*(-1), 0)
		parsed = true
	}
	if strings.Contains(rest, "d") {
		durs := strings.Split(rest, "d")
		daysstring := durs[0]
		if len(durs) >= 2 {
			rest = strings.Join(durs[1:], "")
		}
		days, err := strconv.Atoi(daysstring)
		if err != nil {
			return err
		}
		date = date.AddDate(0, 0, days*(-1))
		parsed = true
	}
	restdur := time.Duration(0)
	if len(rest) > 0 {
		dur, err := time.ParseDuration(rest)
		if err != nil {
			return err
		}
		restdur = dur
		parsed = true
	}

	if parsed {
		f.LastActivityDuration = time.Since(date)
		f.LastActivityDuration += restdur
	}

	return nil
}

func resolveToken(tokenString string, tokenFile string) (string, error) {
	if tokenString == "" && tokenFile == "" {
		return "", nil
	}
	if tokenString != "" {
		envstring := os.Getenv(tokenString)
		if envstring != "" {
			return envstring, nil
		}
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

// Repo TODO.
type Repo struct {
	Name          string
	URL           string
	SSHURL        string
	Token         string
	Defaultbranch string
	Origin        GenRepo
	Owner         string
	Hoster        string
}

// Site TODO.
type Site struct {
	URL  string
	User string
	Port int
}

// GetHost TODO.
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

// GetValues TODO.
func (s *Site) GetValues(url string) error {
	if strings.HasPrefix(url, "ssh://") {
		url = strings.Split(url, "ssh://")[1]
		userurl := strings.Split(url, "@")
		s.User = userurl[0]

		urlport := strings.Split(userurl[1], ":")
		s.URL = urlport[0]

		portstring := strings.Split(urlport[1], "/")[0]

		port, err := strconv.Atoi(portstring)
		if err != nil {
			return err
		}

		s.Port = port

		return nil
	}

	userurl := strings.Split(url, "@")
	s.User = userurl[0]

	urlport := strings.Split(userurl[1], ":")
	s.URL = urlport[0]
	s.Port = 22

	return nil
}

var (
	// Red Render message with Red color.
	Red = color.FgRed.Render
	// Green Render message with Green color.
	Green = color.FgGreen.Render
	// Blue Render message with Blue color.
	Blue = color.FgBlue.Render
	// DotGitRx .git regexp.
	DotGitRx = regexp.MustCompile(`\.git$`)
)

// GetMap TODO.
func GetMap(excludes []string) map[string]bool {
	excludemap := make(map[string]bool)
	for _, exclude := range excludes {
		excludemap[exclude] = true
	}

	return excludemap
}

func statRemoteSSH(sshURL string, repo GenRepo) (string, transport.AuthMethod, error) {
	url := DotGitRx.ReplaceAllString(sshURL, ".wiki.git")

	if repo.SSHKey == "" {
		home := os.Getenv("HOME")
		repo.SSHKey = path.Join(home, ".ssh", "id_rsa")
	}

	auth, err := ssh.NewPublicKeysFromFile("git", repo.SSHKey, "")

	return url, auth, err
}

// StatRemote TODO.
func StatRemote(remoteURL, sshURL string, repo GenRepo) bool {
	var (
		url  string
		auth transport.AuthMethod
		err  error
	)

	if repo.SSH {
		url, auth, err = statRemoteSSH(sshURL, repo)
		if err != nil {
			return false
		}
	} else {
		url = DotGitRx.ReplaceAllString(remoteURL, ".wiki.git")
		if repo.Token != "" {
			auth = &http.BasicAuth{
				Username: "xyz",
				Password: repo.Token,
			}
		} else if repo.Username != "" && repo.Password != "" {
			auth = &http.BasicAuth{
				Username: repo.Username,
				Password: repo.Password,
			}
		}
	}

	remoteConfig := config.RemoteConfig{
		Name: "origin",
		URLs: []string{url},
	}

	_, err = git.NewRemote(nil, &remoteConfig).List(&git.ListOptions{Auth: auth})

	return err == nil
}
