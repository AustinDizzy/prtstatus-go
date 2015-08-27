package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/mirror/v1"
	"gopkg.in/pg.v3"
)

var dir string

func init() {
	dir, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	c, err := ioutil.ReadFile(dir + "/config.json")
	err = json.Unmarshal(c, &config)

	if err != nil {
		log.Printf("Error loading config file: %v\n", err)
	} else {
		log.Printf("Loaded config: %#v\n", config)
	}

	DB = pg.Connect(&pg.Options{
		User:     config.Postgres.User,
		Database: config.Postgres.DB,
	})

	oauthConfig = &oauth2.Config{
		ClientID:     config.OAuthConfig.ClientId,
		ClientSecret: config.OAuthConfig.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{mirror.GlassTimelineScope},
	}
}

func LogErr(err error, args ...interface{}) {
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])

	if config.Debug {

		log.Printf("RUNNING %s", f.Name())

		if len(args) > 0 {
			for i := range args {
				log.Printf("%#v", args[i])
			}
		}
	}
	if err != nil {
		log.Println(err)
	}
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
	router.HandleFunc("/user", UserHandler).Methods("POST")
	router.HandleFunc("/auth", AuthHandler).Methods("GET")
	router.HandleFunc("/store", CallbackHandler)
	router.HandleFunc("/api", ApiRoot)
	router.HandleFunc("/api/", ApiRoot)
	router.HandleFunc("/api/{action}", ApiHandler)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(dir + "/static")))

	log.Println("Now listening on port", config.Port, "serving from", dir+"/static")
	http.ListenAndServe(config.Port, LoggingMiddleware(router))
}

func GetPRT() {

	var url string

	if strings.Contains(config.DataURL, "{TIMESTAMP}") {
		t := strconv.FormatInt(time.Now().Unix(), 10)
		url = strings.Replace(config.DataURL, "{TIMESTAMP}", t, 1)
	} else {
		url = config.DataURL
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cache-Control", "max-age=0")
	res, err := client.Do(req)
	LogErr(err, "making request")

	if res != nil {

		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)

		var data PRTStatus
		err = json.Unmarshal(body, &data)
		LogErr(err, "parsing data")

		lastStatus, err := getLastData()

		if !data.compare(lastStatus) {
			log.Println("PRT update at", time.Now())
			storeData(&data)

			if config.IsLive {
				users, err := getUsers("android")
				LogErr(err, "getting users ", len(users))
				err = sendToUser(&data, users...)
				LogErr(err, "send update to", len(users), "users")
			} else {
				log.Println("AlertUsers(): ", data)
			}
		}

	} else {
		log.Println("Handled http.Client or http.Transport error.")
	}
}

//func SendCard(tokens *oauth.Token, message string) {
//	t := &oauth.Transport{Token: tokens}
//	oauthHttpClient := t.Client()
//
//	mirrorService, err := mirror.New(oauthHttpClient)
//	LogErr(err, "sending Glass card")
//	card := &mirror.TimelineItem{
//		Text:         message,
//		MenuItems:    []*mirror.MenuItem{&mirror.MenuItem{Action: "DELETE"}, &mirror.MenuItem{Action: "TOGGLE_PINNED"}, &mirror.MenuItem{Action: "READ_ALOUD"}},
//		Notification: &mirror.NotificationConfig{Level: "DEFAULT"},
//	}
//	mirrorService.Timeline.Insert(card).Do()
//}
