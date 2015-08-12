package main

import (
	"log"
	"net/http"
	"strconv"

	"code.google.com/p/goauth2/oauth"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	count := strconv.Itoa(UserCount())
	w.Write([]byte("{\"message\": \"PRT API Endpoint\", \"users\": " + count + ", \"success\": true}"))
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Println("Incoming request")
	r.ParseForm()
	body := make(map[string][]string)
	body["regID"] = r.PostForm["regID"]
	if UserStore(body["regID"][0], &oauth.Token{}, "android") {
		w.Write([]byte("{\"success\": true}"))
		log.Println("Added user")
		go InitPush(body["regID"])
	} else {
		w.Write([]byte("{\"success\": false, \"message\": \"User already exists\"}"))
	}
}

func AuthHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, oauthConfig.AuthCodeURL(""), http.StatusFound)
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {

	code := r.FormValue("code")

	t := &oauth.Transport{Config: oauthConfig}
	tokens, _ := t.Exchange(code)

	if UserStore("", tokens, "glass") {
		w.Write([]byte("PRT Status has been successfully enabled on your Google Glass device. We've sent a test message just to make sure. You may now close this page."))
		SendCard(tokens, "PRT Status has been successfully enabled on your Google Glass device.")
	} else {
		w.Write([]byte("ERROR: Something seems to have gone wrong somewhere. If you keep experiencing this issue, please contact me at hi@austindizzy.me."))
	}
}
