package main

// Gitea
type Gitea struct {
	GenRepo
}

// Destination
type Destination struct {
	Gitlab []Gitlab `yaml:"gitlab"`
	Local  []Local  `yaml:"local"`
	Github []Github `yaml:"github"`
	Gitea  []Gitea  `yaml:"gitea"`
	Gogs   []Gogs   `yaml:"gogs"`
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
	Gogs   []Gogs   `yaml:"gogs"`
	Gitlab []Gitlab `yaml:"gitlab"`
	Github []Github `yaml:"github"`
	Gitea  []Gitea  `yaml:"gitea"`
}

// Generell Repo
type GenRepo struct {
	Github
	Url string `yaml:"url"`
}

// Gogs
type Gogs struct {
	GenRepo
}

// Gitlab
type Gitlab struct {
	GenRepo
}

// Github
type Github struct {
	Token    string `yaml:"token"`
	User     string `yaml:"user"`
	SSH      bool   `yaml:"ssh"`
	SSHKey   string `yaml:"sshkey"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Repo
type Repo struct {
	Name          string
	Url           string
	SshUrl        string
	Token         string
	Defaultbranch string
	Origin        Github
}
