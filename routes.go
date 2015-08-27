package main

import (
	"encoding/json"
	"github.com/fatih/structs"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
	"time"
)

func LoggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL)
		h.ServeHTTP(w, r)
	})
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Println("Incoming request")
	r.ParseForm()
	user := User{
		RegistrationId: r.PostForm["regID"][0],
		Device:         "android",
	}
	if err := storeUser(&user); err != nil {
		w.Write([]byte("{\"success\": false, \"message\": \"User already exists\"}"))
	} else {
		w.Write([]byte("{\"success\": true}"))
		var data PRTStatus
		_, err := DB.QueryOne(&data, `SELECT * FROM updates ORDER BY id DESC LIMIT 1`)
		LogErr(err, "get last update")
		sendToUser(&data, user)
	}
}

func AuthHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, oauthConfig.AuthCodeURL(""), http.StatusFound)
}

func ApiRoot(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"message": "PRT Status endpoint. Read more here: https://github.com/AustinDizzy/prtstatus-su",
		"users":   userCount(),
		"success": true,
	}
	json.NewEncoder(w).Encode(data)

}

func ApiHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")
	data := map[string]interface{}{}

	if config.Debug {
		data["route"] = vars["action"]
	}

router:
	switch vars["action"] {
	case "data":
		d := []time.Duration{}
		for _, v := range strings.Split(r.FormValue("range"), "...") {
			bound, err := time.ParseDuration(v)
			if err != nil {
				data["success"] = false
				data["message"] = "The supplied range is in an incorrect format."
				break router
			}
			d = append(d, bound)
		}
		updates, err := getData(d...)
		if err != nil {
			data["success"] = false
			data["message"] = err.Error
		}

		results := make([]map[string]interface{}, len(updates))
		for i, s := range updates {
			results[i] = structs.New(s).Map()
		}
		data["results"] = results
		data["success"] = true
	default:
		data["message"] = "This route does not exist."
		data["success"] = false
	}
	json.NewEncoder(w).Encode(data)
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {

	code := r.FormValue("code")
	LogErr(nil, "glass callback", code)
	//t := &oauth.Transport{Config: oauthConfig}
	//tokens, _ := t.Exchange(code)
	//
	//if UserStore("", tokens, "glass") {
	//	w.Write([]byte("PRT Status has been successfully enabled on your Google Glass device. We've sent a test message just to make sure. You may now close this page."))
	//	SendCard(tokens, "PRT Status has been successfully enabled on your Google Glass device.")
	//} else {
	w.Write([]byte("ERROR: Something seems to have gone wrong somewhere. If you keep experiencing this issue, please contact me at hi@austindizzy.me."))
	//}
}
