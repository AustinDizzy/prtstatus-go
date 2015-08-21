package main

import (
	"log"
	"net/http"
	"strconv"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	count := strconv.Itoa(userCount())
	w.Write([]byte("{\"message\": \"PRT API Endpoint\", \"users\": " + count + ", \"success\": true}"))
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
