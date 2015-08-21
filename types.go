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

func (users *Users) NewRecord() interface{} {
	users.C = append(users.C, User{})
	return &users.C[len(users.C)-1]
}

type UserIDs struct {
	C []string
}

func (userIDs *UserIDs) NewRecord() interface{} {
	userIDs.C = append(userIDs.C, "")
	return &userIDs.C[len(userIDs.C)-1]
}

type PRTStatus struct {
	Id        int    `structs:"-"`
	Status    string `structs:"status"`
	Message   string `structs:"message"`
	Timestamp string `structs:"timestamp"`
	Stations  []struct {
		Id   int
		Name string
	} `structs:"stations"`
	BussesDispatched string `structs:"bussesDispatched"`
	data             string `structs:"-"`
}

func (p *PRTStatus) getStations() []string {
	s := []string{}
	for i := range p.Stations {
		s = append(s, p.Stations[i].Name)
	}
	return s
}

func (p *PRTStatus) bussesRunning() bool {
	return (p.BussesDispatched != "0")
}

type Statuses struct {
	C []PRTStatus
}

func (statuses *Statuses) NewRecord() interface{} {
	statuses.C = append(statuses.C, PRTStatus{})
	return &statuses.C[len(statuses.C)-1]
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
