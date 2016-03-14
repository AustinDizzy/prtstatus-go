package main

import (
	"encoding/json"
	"fmt"
	"github.com/austindizzy/prtstatus-go/prt"
	"github.com/spf13/viper"
	"gopkg.in/macaron.v1"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Response map[string]interface{}

func pbUserAuth(ctx *macaron.Context) {
	if ctx.Query("error") == "access_denied" {
		ctx.Data["Error"] = "access_denied"
		ctx.HTML(200, "pushbullet")
		return
	}

	if ctx.Query("s") == "auth_step" {
		var (
			data = url.Values{
				"grant_type":    {"authorization_code"},
				"client_id":     {viper.GetString("pushbullet.client_id")},
				"client_secret": {viper.GetString("pushbullet.client_secret")},
				"code":          {ctx.Query("code")},
			}
			resp, err = http.PostForm("https://api.pushbullet.com/oauth2/token", data)
			body      []byte
			pbResp    map[string]string
		)

		if err != nil || resp.StatusCode != http.StatusOK {
			log(err, ctx.Data)
			ctx.Data["Error"] = "gen_error"
			ctx.HTML(500, "pushbullet")
			return
		}

		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log(err)
			ctx.Data["Error"] = "gen_error"
			ctx.HTML(500, "pushbullet")
			return
		}

		err = json.Unmarshal(body, &pbResp)

		if len(pbResp["access_token"]) > 0 {
			u := NewUser(pbResp["access_token"], "pushbullet")
			err = storeUser(u)
			if err != nil {
				log(err)
				ctx.Data["Error"] = "gen_error"
				ctx.HTML(500, "pushbullet")
				return
			}

			lastStatus, err := getLastStatus()
			if err == nil {
				err = u.send(&lastStatus)
				if err == nil {
					ctx.Data["Error"] = "no_send_err"
				}
			}
		} else {
			ctx.Data["Error"] = "gen_error"
			ctx.HTML(200, "pushbullet")
		}

		ctx.HTML(200, "pushbullet")
		return
	}

	pushbulletUrl, _ := url.Parse("https://www.pushbullet.com/authorize")
	pushbulletQ := pushbulletUrl.Query()
	pushbulletQ.Add("client_id", viper.GetString("pushbullet.client_id"))
	pushbulletQ.Add("response_type", "code")

	callbackUrl, _ := url.Parse(viper.GetString("pushbullet.redirect_uri"))
	callbackQ := callbackUrl.Query()
	callbackQ.Add("s", "auth_step")
	callbackUrl.RawQuery = callbackQ.Encode()
	pushbulletQ.Add("redirect_uri", callbackUrl.String())

	pushbulletUrl.RawQuery = pushbulletQ.Encode()

	ctx.Redirect(pushbulletUrl.String())
}

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

func indexHandler(ctx *macaron.Context) {
	ctx.HTML(200, "index")
}

func dataHandler(ctx *macaron.Context) {
	ctx.HTML(200, "data")
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
