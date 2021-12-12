package types

import (
	"strconv"
	"strings"

	"github.com/gookit/color"
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

func GetExcludedMap(excludes []string) map[string]bool {
	excludemap := make(map[string]bool)
	for _, exclude := range excludes {
		excludemap[exclude] = true
	}
	return excludemap
}
