package main

// Gitea
type Gitea struct {
	Token string `yaml:"token"`
	User  string `yaml:"user"`
	Url   string `yaml:"url"`
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

// Gogs
type Gogs struct {
	Token string `yaml:"token"`
	User  string `yaml:"user"`
	Url   string `yaml:"url"`
}

// Gitlab
type Gitlab struct {
	Token string `yaml:"token"`
	User  string `yaml:"user"`
	Url   string `yaml:"url"`
}

// Github
type Github struct {
	Token string `yaml:"token"`
	User  string `yaml:"user"`
}

// Repo
type Repo struct {
	Name          string
	Url           string
	Token         string
	Defaultbranch string
}
