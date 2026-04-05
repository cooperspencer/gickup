package sourcehut

import "time"

type entity struct {
	CanonicalName string `json:"canonicalName"`
}

type repository struct {
	ID          int       `json:"id"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	Name        string    `json:"name"`
	Owner       entity    `json:"owner"`
	Description string    `json:"description"`
	Visibility  string    `json:"visibility"`
}

type repositoryCursor struct {
	Results []repository `json:"results"`
	Cursor  *string      `json:"cursor"`
}

type user struct {
	CanonicalName string           `json:"canonicalName"`
	Username      string           `json:"username"`
	Repository    *repository      `json:"repository"`
	Repositories  repositoryCursor `json:"repositories"`
}

type queryMe struct {
	Me user `json:"me"`
}

type queryUser struct {
	User *user `json:"user"`
}

type mutationCreateRepository struct {
	CreateRepository *repository `json:"createRepository"`
}
