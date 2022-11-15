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
