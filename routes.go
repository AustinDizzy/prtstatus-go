package main

import (
	"fmt"
	"github.com/austindizzy/prtstatus-go/prt"
	"net/http"
	"strconv"
	"time"
)

type Response map[string]interface{}

func userHandler(r *http.Request) Response {
	var (
		user = NewUser(r.PostFormValue("regID"), "android")
		n    int
		err  error
	)

	_, err = DB.QueryOne(&n, `SELECT count(*)::int FROM users WHERE key = ?`, user.Key)
	log(err)

	if n != 0 {
		return Response{"success": false, "message": "User already exists."}
	}

	log(storeUser(user), "Saving New User")

	status, err := getLastStatus()
	log(err)

	log(user.send(&status))

	return Response{"success": true}
}

func dataAPI(r *http.Request) Response {
	var (
		err       error
		params    []interface{}
		statuses  []prt.Status
		timeframe = [2]time.Time{time.Time{}, time.Now()}
		q         = `SELECT status, message, timestamp, stations, busses_dispatched, duration FROM updates`
	)
	const errMsg = "Parameter '%s' is not formatted correctly."

	for i, c := range []string{"from", "to"} {
		timestamp, err := strconv.ParseInt(r.URL.Query().Get(c), 10, 64)
		if err != nil {
			return Response{"success": false, "message": fmt.Sprintf(errMsg, c)}
		}
		timeframe[i] = time.Unix(timestamp, 0)
	}

	params = append(params, timeframe[0], timeframe[1])

	if r.URL.Query().Get("limit") != "" {
		limit, err := strconv.ParseUint(r.URL.Query().Get("limit"), 0, 64)
		if err != nil {
			return Response{"success": false, "message": fmt.Sprintf(errMsg, "limit")}
		}
		q += ` LIMIT ?`
		params = append(params, limit)
	}

	q += ` WHERE timetstamp between ? AND ?`
	_, err = DB.Query(&statuses, q, params...)
	return Response{"success": err == nil, "results": &statuses}
}

func APIMiddleware(h func(r *http.Request) Response) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method == "POST" {
			log(r.ParseForm())
		}
		fmt.Fprint(w, h(r))
	})
}
