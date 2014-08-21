package main

import (
	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/mirror/v1"
	"gopkg.in/mgo.v2"
	"time"
)

type User struct {
	RegistrationID   string       `json:"registrationID,omitempty"`
	Tokens           *oauth.Token `json:"tokens,omitempty"`
	UserDevice       string
	RegistrationDate time.Time
}

type PRTStatus struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type GCMWrapper struct {
	RegistrationIDs []string  `json:"registration_ids"`
	Payload         PRTStatus `json:"data"`
}

type Config struct {
	GCMKey  string
	Port    string
	DataURL string
	MongoDB struct {
		ConnURL          string
		RootDB           string
		UserCollection   string
		StatusCollection string
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
}

var (
	config      *Config
	Session     *mgo.Session
	oauthConfig = &oauth.Config{
		Scope:    mirror.GlassTimelineScope,
		AuthURL:  "https://accounts.google.com/o/oauth2/auth",
		TokenURL: "https://accounts.google.com/o/oauth2/token",
	}
)
