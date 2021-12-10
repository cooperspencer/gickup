package main

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

type versionFlag bool
type dryrunFlag bool
