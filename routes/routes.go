package routes

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/AustinDizzy/prtstatus-go/prt"
	"github.com/AustinDizzy/prtstatus-go/user"
	"github.com/AustinDizzy/prtstatus-go/utils"
	"github.com/qedus/nds"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func User(w http.ResponseWriter, r *http.Request) {
	var (
		c    = appengine.NewContext(r)
		k    = datastore.NewKey(c, "user", r.FormValue("regID"), 0, nil)
		u    = user.NewUser(r.FormValue("regID"))
		err  = datastore.Get(c, k, u)
		data = make(map[string]interface{})
	)
	if err == datastore.ErrNoSuchEntity {
		u.Token = r.FormValue("regID")
		u.Device = "android"
		log.Infof(c, "User-Agent is: %s", r.Header.Get("User-Agent"))
		k, err = datastore.Put(c, k, u)
	}
	data["success"] = (err == nil)
	data["user"] = u
	err = utils.WriteJSON(w, data)
	if err != nil {
		log.Errorf(c, "error writing json for request (/user?id=%s)", r.FormValue("regID"))
	}
}

var upStatus = prt.Status{
	Message: "The PRT is running on a normal schedule.",
	Status:  1,
}

var closedStatus = prt.Status{
	Message: "The PRT is closed.",
	Status:  7,
}

var downStatus = prt.Status{
	Message:  "The PRT is down at all stations. The PRT will be back in service in under 45 mins. Buses will continue to run 15 mins after the PRT is back in service.",
	Status:   5,
	Stations: []string{"All"},
}

func PollWeather(w http.ResponseWriter, r *http.Request) {
	var (
		c            = appengine.NewContext(r)
		key          = datastore.NewKey(c, "weather", "current", 0, nil)
		weather, err = utils.RetrieveWeather(c)
	)
	if err != nil {
		log.Errorf(c, "RetrieveWeather: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	key, err = nds.Put(c, key, weather)
	if err != nil {
		log.Errorf(c, "StoreWeather: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func PollStatus(w http.ResponseWriter, r *http.Request) {
	var (
		c         = appengine.NewContext(r)
		urlClient = urlfetch.Client(c)
		prtClient = prt.NewClient(urlClient)
		forceType = r.URL.Query().Get("force_type")
		noNotif   = r.URL.Query().Get("no_notif")
	)

	status, str, err := prtClient.GetCurrentStatus()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	lastStatus, err := utils.GetCurrentStatus(c)
	switch err {
	case datastore.ErrNoSuchEntity:
		err = utils.StoreNewStatus(c, status)
		if err != nil {
			log.Errorf(c, "StoreNewStatus: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		break
	case nil:
		if len(forceType) > 0 {
			if forceType == "-1" {
				status = &downStatus
			} else if forceType == "0" {
				status = &closedStatus
			} else if forceType == "1" {
				status = &upStatus
			}
			status.Timestamp = lastStatus.Timestamp + 142
		}

		i := lastStatus.CompareTo(status)
		if i == 1 {
			log.Infof(c, "updates maybe stale? current status came before last status: %#v | %#v", lastStatus, status)
			break
		} else if i == -1 {
			log.Infof(c, str)
			err = utils.StoreNewStatus(c, status)
			if err != nil {
				log.Errorf(c, "StoreNewStatus: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			weather, err := utils.GetCurrentWeather(c)
			if err != nil {
				log.Errorf(c, "GetCurrentWeather: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if err = weather.Save(c, status.Timestamp); err != nil {
				log.Errorf(c, "weather.Save: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if !strings.Contains(noNotif, "fcm") {
				rsp, err := utils.NotifyFCM(c, status, weather)
				if err != nil {
					log.Errorf(c, "NotifyFCM: %s", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				log.Infof(c, "sent %#v", rsp)
			}

			if !strings.Contains(noNotif, "pb") {
				err = utils.NotifyPushbullet(c, status)
				if err != nil {
					log.Errorf(c, "NotifyPushbullet: %s", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}
		break
	default:
		log.Errorf(c, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func GetStatus(w http.ResponseWriter, r *http.Request) {
	var (
		c        = appengine.NewContext(r)
		limit    = r.URL.Query().Get("limit")
		num, err = strconv.Atoi(limit)
		q        = datastore.NewQuery("updates").Order("-__key__")
		data     interface{}
		status   *prt.Status
		statuses []prt.Status
	)
	if len(limit) > 0 && err != nil {
		log.Errorf(c, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
	if num < 0 {
		num *= 1
	}
	if num > 25 {
		num = 25
	}
	if num > 1 {
		_, err = q.Limit(num+1).GetAll(c, &statuses)
		data = statuses[1:]
	} else {
		status, err = utils.GetCurrentStatus(c)
		if err != nil {
			log.Errorf(c, err.Error())
		}
		data = status
	}

	if err = utils.WriteJSON(w, data); err != nil {
		log.Errorf(c, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func GetWeather(w http.ResponseWriter, r *http.Request) {
	var (
		c            = appengine.NewContext(r)
		weather, err = utils.GetCurrentWeather(c)
	)
	if err != nil {
		log.Errorf(c, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	} else if err = utils.WriteJSON(w, weather); err != nil {
		log.Errorf(c, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func GetLinks(w http.ResponseWriter, r *http.Request) {
	var (
		linksFile []byte
		err       error
		c         = appengine.NewContext(r)
	)
	if appengine.IsDevAppServer() {
		linksFile, err = ioutil.ReadFile(path.Join(os.Getenv("PWD"), "links.json"))
		if err != nil {
			log.Errorf(c, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		storageClient, err := storage.NewClient(c)
		if err != nil {
			log.Errorf(c, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		bucket, _ := file.DefaultBucketName(c)
		rc, err := storageClient.Bucket(bucket).Object("links.json").NewReader(c)
		if err != nil {
			log.Errorf(c, "error reading links: %v", err.Error())
		}

		defer rc.Close()
		if linksFile, err = ioutil.ReadAll(rc); err != nil {
			log.Errorf(c, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(linksFile)
}
