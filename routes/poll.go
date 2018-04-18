package routes

import (
	"net/http"
	"strings"

	"github.com/AustinDizzy/prtstatus-go/prt"
	"github.com/AustinDizzy/prtstatus-go/utils"
	"github.com/qedus/nds"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

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

// PollWeather polls for weather condition updates and stores any new records
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

// PollStatus polls for PRT status updates and stores any new records
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
