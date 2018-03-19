package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gopkg.in/maddevsio/fcm.v1"

	"github.com/AustinDizzy/prtstatus-go/config"
	"github.com/AustinDizzy/prtstatus-go/prt"
	"github.com/qedus/nds"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/urlfetch"
)

const (
	// FCMTopic is a string of the FCM message topic the notification is pushed to
	FCMTopic = "/topics/prt-updates"
	// pbAPI is the base API endpoint to use for interacting with Pushbullet
	pbAPI = "https://api.pushbullet.com/v2/pushes"
)

func StoreNewStatus(c context.Context, status *prt.Status) error {
	key := datastore.NewKey(c, "updates", "current", 0, nil)
	key, err := nds.Put(c, key, status)
	if err != nil {
		return err
	}

	statusKey := datastore.NewKey(c, "updates", "", status.Timestamp, nil)
	statusKey, err = datastore.Put(c, statusKey, status)
	if err != nil {
		return err
	}

	return nil
}

func NotifyFCM(c context.Context, status *prt.Status) (fcm.Response, error) {
	var (
		cfg, err  = config.Load(c)
		urlClient = urlfetch.Client(c)
		fcmClient = fcm.NewFCMWithClient(cfg["fcmKey"], urlClient)
		msg       = fcm.Message{
			Data: status.ToMap(),
			To:   FCMTopic,
		}
	)
	if err != nil {
		return fcm.Response{}, err
	}
	return fcmClient.Send(msg)
}

func NotifyPushbullet(c context.Context, status *prt.Status) error {
	var (
		cfg, err  = config.Load(c)
		urlClient = urlfetch.Client(c)
		data, _   = json.Marshal(map[string]string{
			"title":       fmt.Sprintf("The PRT is %s", status.GetStatusText()),
			"body":        status.Message,
			"type":        "note",
			"channel_tag": "wvuprtstatus",
		})
	)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", pbAPI, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Access-Token", cfg["pbKey"])
	_, err = urlClient.Do(req)
	return err
}

func WriteJSON(w http.ResponseWriter, data interface{}) error {
	enc := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return enc.Encode(data)
}
