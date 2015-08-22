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

func init() {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
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
	log.Printf("RUNNING %s", f.Name())

	if config.Debug && len(args) > 0 {
		for i := range args {
			log.Printf("%#v", args[i])
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

func compare(a, b PRTStatus) bool {
	LogErr(nil, "comparing", a, b)
	return (a.Status == b.Status &&
		a.Message == b.Message &&
		a.Timestamp == b.Timestamp)
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

		var data, lastStatus PRTStatus
		err = json.Unmarshal(body, &data)
		_, err = DB.QueryOne(&lastStatus, `SELECT * FROM updates ORDER BY id DESC LIMIT 1`)

		if !compare(data, lastStatus) {
			log.Println("PRT update at", time.Now())

			_, err = DB.QueryOne(&data, `
					INSERT INTO updates (status, message, timestamp, stations, bussesDispatched, data)
					VALUES (?, ?, ?, ?, ?, ?)
					RETURNING id
			`, data.Status, data.Message, data.Timestamp, data.getStations(), data.bussesRunning(), string(body))

			LogErr(err, "inserting data")

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
