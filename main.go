package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/gorilla/mux"
	"google.golang.org/api/mirror/v1"
	"gopkg.in/pg.v3"
)

func init() {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	c, err := ioutil.ReadFile(dir + "/config.json")
	err = json.Unmarshal(c, &config)
	LogErr(err, "config reading")

	DB = pg.Connect(&pg.Options{
		User:     config.Postgres.User,
		Database: config.Postgres.DB,
	})

	oauthConfig.ClientId = config.OAuthConfig.ClientId
	oauthConfig.ClientSecret = config.OAuthConfig.ClientSecret
	oauthConfig.RedirectURL = config.OAuthConfig.RedirectURL
	oauthConfig.ApprovalPrompt = config.OAuthConfig.ApprovalPrompt
	oauthConfig.AccessType = config.OAuthConfig.AccessType
}

func LogErr(err error, args ...interface{}) {
	if config.Debug && cap(args) > 0 {
		log.Printf("", args)
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
		_, err = DB.QueryOne(&lastStatus, `
	    SELECT * FROM updates ORDER BY id DESC LIMIT 1
		`)

		if !compare(data, lastStatus) {
			log.Println("PRT update at ", time.Now())

			data.bussesBool = data.BussesDispatched != "0"

			_, err = DB.QueryOne(&data, `
					INSERT INTO updates (status, message, timestamp, stations, bussesDispatched, data)
					VALUES (?, ?, ?, ?, ?, ?)
					RETURNING id
			`, data.Status, data.Message, data.Timestamp, data.Stations, data.bussesBool, string(body))

			LogErr(err, "inserting data")

			if config.IsLive {
				//go AlertUsers(data)
				log.Println("users would be alerted")
			} else {
				log.Println("AlertUsers(): ", data)
			}
		}

	} else {
		log.Println("Handled http.Client or http.Transport error.")
	}
}

func InitPush(user []string) {
	url := "https://android.googleapis.com/gcm/send"

	payload := PRTStatus{}
	_, err := DB.QueryOne(&payload, `
		SELECT * FROM updates ORDER BY id DESC LIMIT 1
	`)

	log.Printf("InitPush err: %v", err)

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
	} else {
		body, err := ioutil.ReadAll(resp.Body)

		log.Println("GCM Results:", string(body))

		var response GCMResult
		err = json.Unmarshal(body, &response)

		if err == nil {

			log.Println("There are", len(androids), "Androids and", len(response.Results), "results.")
			log.Println("There are", response.Success, "successful messages,", response.Failure, "failures, and", response.CanonicalIDs, "canonical IDs.")
			mappedResults := make(map[string]GCMInnerResults)
			for i, v := range androids {
				mappedResults[v] = response.Results[i]
			}

			notRegisteredCount := 0
			canonicalIdCount := 0

			for i, v := range mappedResults {
				if v.Error == "NotRegistered" {
					notRegisteredCount++
					go DeleteUser(i)
				}

				if len(v.RegistrationID) > 0 {
					canonicalIdCount++
					go UpdateUser(i, v.RegistrationID)
				}
			}

			log.Println("There are", notRegisteredCount, " NotRegistered and", canonicalIdCount, " Canonical ID updates after result mapping.")
		}
	}

	var glass []*oauth.Token
	glass = GetGlass()
	for i := range glass {
		SendCard(glass[i], payload.Message)
	}
}

func UpdateUser(oldId string, newId string) {

	log.Printf("UpdateUser: %v -> %v\n", oldId, newId)
	// c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	// err := c.Update(bson.M{"registrationid": oldId}, bson.M{"$set": bson.M{"registrationid": newId}})
	//
	// PanicErr(err)
	// if err == nil {
	// 	log.Println("User", oldId, "was updated to user", newId, "successfully.")
	// }
}

func DeleteUser(userId string) {

	log.Printf("DeleteUser: %v\n", userId)
	// session := Session.Clone()
	// defer session.Close()
	//
	// c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	// err := c.Remove(bson.M{"registrationid": userId})
	//
	// PanicErr(err)
	// if err == nil {
	// 	log.Println("User", userId, "unregistered and was deleted.")
	// }
}

func GetAndroid() []string {
	var result []struct{ RegistrationID string }

	// session := Session.Clone()
	// defer session.Close()
	//
	// c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	// iter := c.Find(bson.M{"userdevice": "android"}).Iter()
	// err := iter.All(&result)
	// PanicErr(err)
	//
	var finalResult []string

	for i := range result {
		finalResult = append(finalResult, result[i].RegistrationID)
	}
	return finalResult
}

func GetGlass() []*oauth.Token {
	var result []struct{ Tokens oauth.Token }

	// session := Session.Clone()
	// defer session.Close()
	//
	// c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	// iter := c.Find(bson.M{"userdevice": "glass"}).Iter()
	// err := iter.All(&result)
	// PanicErr(err)

	var finalResult []*oauth.Token

	for i := range result {
		finalResult = append(finalResult, &result[i].Tokens)
	}
	return finalResult
}

func SendCard(tokens *oauth.Token, message string) {
	t := &oauth.Transport{Token: tokens}
	oauthHttpClient := t.Client()

	mirrorService, err := mirror.New(oauthHttpClient)
	LogErr(err, "sending Glass card")
	card := &mirror.TimelineItem{
		Text:         message,
		MenuItems:    []*mirror.MenuItem{&mirror.MenuItem{Action: "DELETE"}, &mirror.MenuItem{Action: "TOGGLE_PINNED"}, &mirror.MenuItem{Action: "READ_ALOUD"}},
		Notification: &mirror.NotificationConfig{Level: "DEFAULT"},
	}
	mirrorService.Timeline.Insert(card).Do()
}

func UserCount() int {
	// session := Session.Clone()
	// defer session.Close()
	//
	// c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	// count, err := c.Count()

	// return count
	return 1
}

func UserStore(regID string, tokens *oauth.Token, device string) bool {

	// session := Session.Clone()
	// defer session.Close()
	//
	// c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	// userExists := User{}
	// err := c.Find(bson.M{"registrationid": regID}).One(&userExists)
	// if err != nil {
	// 	if device == "android" {
	// 		err = c.Insert(&User{
	// 			RegistrationID:   regID,
	// 			UserDevice:       "android",
	// 			RegistrationDate: time.Now(),
	// 		})
	// 	} else {
	// 		err = c.Insert(&User{
	// 			Tokens:           tokens,
	// 			UserDevice:       "glass",
	// 			RegistrationDate: time.Now(),
	// 		})
	// 	}
	// 	PanicErr(err)
	// } else {
	// 	return false
	// }

	return true
}
