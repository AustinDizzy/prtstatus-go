package routes

import (
	"net/http"
	"strings"

	"github.com/AustinDizzy/prtstatus-go/prt"
	"github.com/AustinDizzy/prtstatus-go/user"
	"github.com/AustinDizzy/prtstatus-go/utils"
	"github.com/qedus/nds"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
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

func Poll(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	var (
		key              = datastore.NewKey(c, "updates", "current", 0, nil)
		lastStatus       = new(prt.Status)
		urlClient        = urlfetch.Client(c)
		prtClient        = prt.NewClient(urlClient)
		status, str, err = prtClient.GetCurrentStatus()
		forceType        = r.URL.Query().Get("force_type")
		noNotif          = r.URL.Query().Get("no_notif")
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = nds.Get(c, key, lastStatus)
	if err == datastore.ErrNoSuchEntity {
		err = utils.StoreNewStatus(c, status)
		if err != nil {
			log.Errorf(c, "%s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		log.Errorf(c, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	} else if err == nil {
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
		switch lastStatus.CompareTo(status) {
		case 1:
			log.Infof(c, "updates maybe stale? current status came before last status: %#v | %#v", lastStatus, status)
			break
		case -1:
			log.Infof(c, str)
			err = utils.StoreNewStatus(c, status)
			if err != nil {
				log.Errorf(c, "%s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if !strings.Contains(noNotif, "fcm") {
				rsp, err := utils.NotifyFCM(c, status)
				if err != nil {
					log.Errorf(c, "%s", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				log.Infof(c, "sent %#v", rsp)
			}

			if !strings.Contains(noNotif, "pb") {
				err = utils.NotifyPushbullet(c, status)
				if err != nil {
					log.Errorf(c, "error pushing to pushbullet: %s", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			break
		case 0:
			// do nothing because statuses are same
		}
	} else {
		log.Errorf(c, "%s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func LastStatus(w http.ResponseWriter, r *http.Request) {
	var (
		c          = appengine.NewContext(r)
		key        = datastore.NewKey(c, "updates", "current", 0, nil)
		lastStatus prt.Status
		err        = nds.Get(c, key, &lastStatus)
	)
	if err != nil {
		log.Errorf(c, "%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err = utils.WriteJSON(w, lastStatus); err != nil {
		log.Errorf(c, "%s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
