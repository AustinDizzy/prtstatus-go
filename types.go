package main

import (
	"time"

	"golang.org/x/oauth2"
	"gopkg.in/pg.v3"
)

type User struct {
	RegistrationId   string
	Tokens           *oauth2.Token
	Device           string
	RegistrationDate time.Time
}

type Users struct {
	C []User
}

var _ pg.Collection = &Users{}

type PRTStatus struct {
	Id           int    `structs:"-"`
	Status       string `structs:"status"`
	Message      string `structs:"message"`
	Timestamp    string `structs:"timestamp"`
	stationsData []struct {
		Id   int
		Name string
	} `json:"stations" structs:"stations" pg:"-"`
	Stations            []string `sql:"stations" pg:"stations" structs:"-"`
	BussesDispatched    bool     `structs:"bussesDispatched" json:"bussesDispatchedBool"`
	bussesDispatchedStr string   `json:"bussesDispatched"`
}

type Updates struct {
	C []PRTStatus
}

type Config struct {
	GCMKey   string
	Port     string
	DataURL  string
	Postgres struct {
		User string
		DB   string
	}
	RefreshInterval string
	IsLive          bool
	OAuthConfig     struct {
		ClientId       string
		ClientSecret   string
		RedirectURL    string
		ApprovalPrompt string
		AccessType     string
	}
	Debug bool
}

var (
	config      *Config
	DB          *pg.DB
	oauthConfig *oauth2.Config
)
