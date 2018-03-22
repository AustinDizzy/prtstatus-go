package main

import (
	"net/http"

	"github.com/AustinDizzy/prtstatus-go/routes"
	"google.golang.org/appengine"
)

func init() {
	http.HandleFunc("/api/user", routes.User)
	http.HandleFunc("/api/status", routes.LastStatus)
	http.HandleFunc("/api/links", routes.GetLinks)
	http.HandleFunc("/api/poll", routes.Poll)
	http.Handle("/", http.FileServer(http.Dir("static")))
}

func main() {
	appengine.Main()
}
