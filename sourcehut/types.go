package sourcehut

import "time"

type Repositories struct {
	Next           string       `json:"next"`
	Results        []Repository `json:"results"`
	Total          int          `json:"total"`
	ResultsPerPage int          `json:"results_per_page"`
}

type Repository struct {
	ID      int       `json:"id"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Name    string    `json:"name"`
	Owner   struct {
		CanonicalName string `json:"canonical_name"`
		Name          string `json:"name"`
	} `json:"owner"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type Refs struct {
	Next           string `json:"next"`
	Results        []Ref  `json:"results"`
	Total          int    `json:"total"`
	ResultsPerPage int    `json:"results_per_page"`
}

type Ref struct {
	Target    string        `json:"target"`
	Name      string        `json:"name"`
	Artifacts []interface{} `json:"artifacts"`
}

type User struct {
	CanonicalName string      `json:"canonical_name"`
	Name          string      `json:"name"`
	Email         string      `json:"email"`
	URL           interface{} `json:"url"`
	Location      interface{} `json:"location"`
	Bio           interface{} `json:"bio"`
}

type Commits struct {
	Next           interface{} `json:"next"`
	Results        []Results   `json:"results"`
	Total          int         `json:"total"`
	ResultsPerPage int         `json:"results_per_page"`
}
type Author struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}
type Committer struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}
type Results struct {
	ID        string      `json:"id"`
	ShortID   string      `json:"short_id"`
	Author    Author      `json:"author"`
	Committer Committer   `json:"committer"`
	Timestamp time.Time   `json:"timestamp"`
	Message   string      `json:"message"`
	Tree      string      `json:"tree"`
	Parents   []string    `json:"parents"`
	Signature interface{} `json:"signature"`
}

type PostRepo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
}
