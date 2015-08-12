package main

import (
	"time"

	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/mirror/v1"
	"gopkg.in/pg.v3"
)

type User struct {
	RegistrationID   string       `json:"registrationID,omitempty"`
	Tokens           *oauth.Token `json:"tokens,omitempty"`
	UserDevice       string
	RegistrationDate time.Time
}

type PRTStatus struct {
	ID               int
	Status           string   `json:"status"`
	Message          string   `json:"message"`
	Timestamp        string   `json:"timestamp"`
	Stations         []string `json:"stations"`
	BussesDispatched string   `json:"bussesDispatched"`
	bussesBool       bool
	data             string
}

type GCMWrapper struct {
	RegistrationIDs []string  `json:"registration_ids"`
	Payload         PRTStatus `json:"data"`
}

type GCMResult struct {
	MulticastID  int64 `json:"multicast_id"`
	Success      int
	Failure      int
	CanonicalIDs int `json:"canonical_ids"`
	Results      []GCMInnerResults
}

type GCMInnerResults struct {
	MessageID      string `json:"message_id"`
	RegistrationID string `json:"registration_id"`
	Error          string
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
	oauthConfig = &oauth.Config{
		Scope:    mirror.GlassTimelineScope,
		AuthURL:  "https://accounts.google.com/o/oauth2/auth",
		TokenURL: "https://accounts.google.com/o/oauth2/token",
	}
)
