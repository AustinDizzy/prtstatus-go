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
	GCMKey  string `json:"GCMKey"`
	Port    string `json:"Port"`
	DataURL string `json:"DataURL"`
	MongoDB struct {
		ConnURL          string `json:"ConnURL"`
		RootDB           string `json:"RootDB"`
		UserCollection   string `json:"UserCollection"`
		StatusCollection string `json:"StatusCollection"`
	}
	RefreshInterval string `json:"RefreshInterval"`
	IsLive          bool   `json:"IsLive"`
	OAuthConfig     struct {
		ClientId       string `json:"ClientId"`
		ClientSecret   string `json:"ClientSecret"`
		RedirectURL    string `json:"RedirectURL"`
		ApprovalPrompt string `json:"ApprovalPrompt"`
		AccessType     string `json:"AccessType"`
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
