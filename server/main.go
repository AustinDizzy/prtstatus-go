package main

import (
	"net/http"

	"github.com/AustinDizzy/prtstatus-go/routes"
	"google.golang.org/appengine"
)

func init() {
	http.HandleFunc("/api/user", routes.User)
	http.HandleFunc("/api/status", routes.GetStatus)
	http.HandleFunc("/api/weather", routes.GetWeather)
	http.HandleFunc("/api/poll/status", routes.PollStatus)
	http.HandleFunc("/api/poll/weather", routes.PollWeather)
	http.Handle("/", http.FileServer(http.Dir("static")))
}

func main() {
	appengine.Main()
}
