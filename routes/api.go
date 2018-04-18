package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/AustinDizzy/prtstatus-go/prt"
	"github.com/AustinDizzy/prtstatus-go/user"
	"github.com/AustinDizzy/prtstatus-go/utils"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

// User is route to query or create new User
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
		u.RegistrationDate = time.Now()
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

// GetStatus is route for returning a status or a list of statuses
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

// GetWeather is route for returning current weather conditions
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
