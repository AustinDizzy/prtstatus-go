package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"os"
	"path/filepath"
)

type User struct {
	RegistrationID   string
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
}

var (
	config  *Config
	Session *mgo.Session
)

func init() {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	c, err := ioutil.ReadFile(dir + "/config.json")
	PanicErr(err)
	err = json.Unmarshal(c, &config)

	Session, err = mgo.Dial(config.MongoDB.ConnURL)
	PanicErr(err)
	Session.SetMode(mgo.Monotonic, true)
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

	http.Handle("/prt/_sandbox/", router)
	http.Handle("/prt/_sandbox/user", router)

	log.Println("Now listening on port", config.Port)
	http.ListenAndServe(config.Port, nil)
}

func PanicErr(err error) {
	if err != nil {
		panic(err)
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
		}
	}
}

func AlertUsers(payload PRTStatus) {
	url := "https://android.googleapis.com/gcm/send"

	client := &http.Client{
		Jar: nil,
	}

	users := GetAllUsers()
	gcmWrapper := &GCMWrapper{RegistrationIDs: users, Payload: payload}
	gcmMessage, _ := json.Marshal(gcmWrapper)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(gcmMessage))
	req.Header.Add("Authorization", "key="+config.GCMKey)
	req.Header.Add("Content-Type", "application/json")
	resp, _ := client.Do(req)
	if resp.StatusCode != 200 {
		log.Println("GCM Failed. Resp: ", resp.StatusCode)
	}
}

func GetAllUsers() []string {
	var result []struct{ RegistrationID string }

	session := Session.Clone()
	defer session.Close()

	c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	iter := c.Find(nil).Iter()
	err := iter.All(&result)
	PanicErr(err)

	var finalResult []string

	for i := range result {
		finalResult = append(finalResult, result[i].RegistrationID)
	}
	return finalResult
}

func UserHandler(respWriter http.ResponseWriter, request *http.Request) {
	respWriter.Header().Set("Content-Type", "application/json")
	request.ParseForm()
	body := make(map[string][]string)
	body["regID"] = request.PostForm["regID"]
	if UserStore(body["regID"][0]) {
		respWriter.Write([]byte("{\"success\": true}"))
	} else {
		respWriter.Write([]byte("{\"success\": false, \"message\": \"User already exists\"}"))
	}
}

func RootHandler(respWriter http.ResponseWriter, request *http.Request) {
	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.Write([]byte("{\"message\": \"PRT API Endpoint\", \"success\": true}"))
}

func UserStore(regID string) bool {

	session := Session.Clone()
	defer session.Close()

	currentTime := time.Now()

	c := session.DB(config.MongoDB.RootDB).C(config.MongoDB.UserCollection)
	userExists := User{}
	err := c.Find(bson.M{"registrationid": regID}).One(&userExists)
	if err != nil {
		err = c.Insert(&User{regID, currentTime})
		PanicErr(err)
	} else {
		return false
	}

	return true
}
