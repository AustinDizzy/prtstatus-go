package main

import (
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/mirror/v1"
	"encoding/json"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

func init() {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	c, err := ioutil.ReadFile(dir + "/config.json")
	PanicErr(err)
	err = json.Unmarshal(c, &config)

	Session, err = mgo.Dial(config.MongoDB.ConnURL)
	PanicErr(err)
	Session.SetMode(mgo.Monotonic, true)

	oauthConfig.ClientId = config.OAuthConfig.ClientId
	oauthConfig.ClientSecret = config.OAuthConfig.ClientSecret
	oauthConfig.RedirectURL = config.OAuthConfig.RedirectURL
	oauthConfig.ApprovalPrompt = config.OAuthConfig.ApprovalPrompt
	oauthConfig.AccessType = config.OAuthConfig.AccessType
}

func main() {
	log.Println("Starting server...")
	duration, _ := time.ParseDuration(config.RefreshInterval)
	ticker := time.NewTicker(duration)
	quit := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				go GetPRT()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/prt/_sandbox/", RootHandler).Methods("GET")
	router.HandleFunc("/prt/_sandbox/user", UserHandler).Methods("POST")
	router.HandleFunc("/prt/_sandbox/auth", AuthHandler).Methods("GET")
	router.HandleFunc("/prt/_sandbox/store", CallbackHandler)

	http.Handle("/prt/_sandbox/", router)
	http.Handle("/prt/_sandbox/user", router)
	http.Handle("/prt/_sandbox/auth", router)
	http.Handle("/prt/_sandbox/store", router)

	log.Println("Now listening on port", config.Port)
	http.ListenAndServe(config.Port, nil)
}

func PanicErr(err error) {
	if err != nil {
		//panic(err)
		//Don't panic until we work out a proper error handling solution.
		//We don't want to crash the whole system again.
		log.Println(err)
	}
}

func GetPRT() {
	url := config.DataURL

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cache-Control", "max-age=0")
	res, err := client.Do(req)
	PanicErr(err)

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	PanicErr(err)

	var data PRTStatus
	err = json.Unmarshal(body, &data)
	PanicErr(err)

	session := Session.Clone()
	defer session.Close()

	c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.StatusCollection)
	existingStatus := PRTStatus{}
	err = c.Find(nil).One(&existingStatus)
	if data != existingStatus {
		log.Println("PRT update at ", time.Now())
		err = c.Update(nil, data)
		PanicErr(err)
		if config.IsLive {
			go AlertUsers(data)
		} else {
			log.Println("AlertUsers(): ", data)
		}
	}
}

func InitPush(user []string) {
	url := "https://android.googleapis.com/gcm/send"

	var payload PRTStatus
	session := Session.Clone()
	defer session.Close()

	c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.StatusCollection)
	c.Find(nil).One(&payload)

	client := &http.Client{
		Jar: nil,
	}
	gcmWrapper := &GCMWrapper{RegistrationIDs: user, Payload: payload}
	gcmMessage, _ := json.Marshal(gcmWrapper)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(gcmMessage))
	req.Header.Add("Authorization", "key="+config.GCMKey)
	req.Header.Add("Content-Type", "application/json")
	resp, _ := client.Do(req)
	if resp.StatusCode != 200 {
		log.Println("GCM Failed. Resp:", resp.StatusCode)
	}
}

func AlertUsers(payload PRTStatus) {
	url := "https://android.googleapis.com/gcm/send"

	client := &http.Client{
		Jar: nil,
	}

	androids := GetAndroid()
	gcmWrapper := &GCMWrapper{RegistrationIDs: androids, Payload: payload}
	gcmMessage, _ := json.Marshal(gcmWrapper)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(gcmMessage))
	req.Header.Add("Authorization", "key="+config.GCMKey)
	req.Header.Add("Content-Type", "application/json")
	resp, _ := client.Do(req)
	if resp.StatusCode != 200 {
		log.Println("GCM Failed. Resp:", resp.StatusCode)
	}

	var glass []*oauth.Token
	glass = GetGlass()
	for i := range glass {
		SendCard(glass[i], payload.Message)
	}
}

func GetAndroid() []string {
	var result []struct{ RegistrationID string }

	session := Session.Clone()
	defer session.Close()

	c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	iter := c.Find(bson.M{"userdevice": "android"}).Iter()
	err := iter.All(&result)
	PanicErr(err)

	var finalResult []string

	for i := range result {
		finalResult = append(finalResult, result[i].RegistrationID)
	}
	return finalResult
}

func GetGlass() []*oauth.Token {
	var result []struct{ Tokens oauth.Token }

	session := Session.Clone()
	defer session.Close()

	c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	iter := c.Find(bson.M{"userdevice": "glass"}).Iter()
	err := iter.All(&result)
	PanicErr(err)

	var finalResult []*oauth.Token

	for i := range result {
		finalResult = append(finalResult, &result[i].Tokens)
	}
	return finalResult
}

func AuthHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, oauthConfig.AuthCodeURL(""), http.StatusFound)
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {

	code := r.FormValue("code")

	t := &oauth.Transport{Config: oauthConfig}
	tokens, _ := t.Exchange(code)

	if UserStore("", tokens, "glass") {
		w.Write([]byte("PRT Status has been successfully enabled on your Google Glass device. We've sent a test message just to make sure. You may now close this page."))
		SendCard(tokens, "PRT Status has been successfully enabled on your Google Glass device.")
	} else {
		w.Write([]byte("ERROR: Something seems to have gone wrong somewhere. If you keep experiencing this issue, please contact me at hi@austindizzy.me."))
	}
}

func SendCard(tokens *oauth.Token, message string) {
	t := &oauth.Transport{Token: tokens}
	oauthHttpClient := t.Client()

	mirrorService, err := mirror.New(oauthHttpClient)
	PanicErr(err)
	card := &mirror.TimelineItem{
		Text:         message,
		MenuItems:    []*mirror.MenuItem{&mirror.MenuItem{Action: "DELETE"}, &mirror.MenuItem{Action: "TOGGLE_PINNED"}, &mirror.MenuItem{Action: "READ_ALOUD"}},
		Notification: &mirror.NotificationConfig{Level: "DEFAULT"},
	}
	mirrorService.Timeline.Insert(card).Do()
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Println("Incoming request")
	r.ParseForm()
	body := make(map[string][]string)
	body["regID"] = r.PostForm["regID"]
	if UserStore(body["regID"][0], &oauth.Token{}, "android") {
		w.Write([]byte("{\"success\": true}"))
		log.Println("Added user")
		go InitPush(body["regID"])
	} else {
		w.Write([]byte("{\"success\": false, \"message\": \"User already exists\"}"))
	}
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{\"message\": \"PRT API Endpoint\", \"success\": true}"))
}

func UserStore(regID string, tokens *oauth.Token, device string) bool {

	session := Session.Clone()
	defer session.Close()

	c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	userExists := User{}
	err := c.Find(bson.M{"registrationid": regID}).One(&userExists)
	if err != nil {
		if device == "android" {
			err = c.Insert(&User{
				RegistrationID:   regID,
				UserDevice:       "android",
				RegistrationDate: time.Now(),
			})
		} else {
			err = c.Insert(&User{
				Tokens:           tokens,
				UserDevice:       "glass",
				RegistrationDate: time.Now(),
			})
		}
		PanicErr(err)
	} else {
		return false
	}

	return true
}
