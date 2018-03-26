package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"gopkg.in/maddevsio/fcm.v1"

	"github.com/AustinDizzy/prtstatus-go/config"
	"github.com/AustinDizzy/prtstatus-go/prt"
	"github.com/qedus/nds"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

const (
	// FCMTopic is a string of the FCM message topic the notification is pushed to
	FCMTopic = "/topics/prt-updates"
	// pbAPI is the base API endpoint to use for interacting with Pushbullet
	pbAPI = "https://api.pushbullet.com/v2/pushes"
	// wunderground API is the API endpoint used to retrieve the weather
	wuAPI = "http://api.wunderground.com/api/%s/conditions/q/pws:KWVMORGA25.json"
)

// Weather contains data from Weather Underground response
type Weather struct {
	// Temperature is the temperature in Fahrenheit
	Temperature float32 `json:"temperature"`
	// Humidity is the relative humidity percentage
	Humidity float32 `json:"humidity"`
	// Weather contains the simple weather conditions and current icon name
	Weather string `json:"weather"`
	// Conditions contains a description of the weather conditions
	Conditions string `json:"conditions"`
	// FeelsLike is the "feels like" temperature in Fahrenheit
	FeelsLike float32 `json:"feelsLike"`
	// Precip1hr is the inches of precipitation from the past hour
	Precip1hr float32 `json:"precip1hr"`
	// PrecipToday is the inches of precipitation today
	PrecipToday float32 `json:"precipToday"`
	// Visibility is the relative kilometers of visibility
	Visibility float32 `json:"visibility"`
	// WindDir is the direction of the wind
	WindDir string `json:"windDir"`
	// WindSpeed is the speed of the wind in miles per hour (MPH)
	WindSpeed float32 `json:"windSpeed"`
}

func RetrieveWeather(c context.Context) (*Weather, error) {
	var (
		data      map[string]interface{}
		resp      *http.Response
		cfg, err  = config.Load(c)
		urlClient = urlfetch.Client(c)
		url       = fmt.Sprintf(wuAPI, cfg["wuKey"])
		weather   = new(Weather)
	)
	if err != nil {
		return weather, err
	}
	resp, err = urlClient.Get(url)
	if err != nil {
		return weather, err
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return weather, err
	}

	if err = json.Unmarshal(buf, &data); err != nil {
		return weather, err
	}

	data = data["current_observation"].(map[string]interface{})
	weather.Temperature = float32(data["temp_f"].(float64))
	humidity, err := strconv.ParseFloat(strings.TrimRight(data["relative_humidity"].(string), "%"), 32)
	if err != nil {
		return weather, err
	}
	weather.Humidity = float32(humidity)
	weather.Weather = data["icon"].(string)
	weather.Conditions = data["weather"].(string)
	feelsLike, err := strconv.ParseFloat(data["feelslike_f"].(string), 32)
	if err != nil {
		return weather, err
	}
	weather.FeelsLike = float32(feelsLike)
	precip1hr, err := strconv.ParseFloat(data["precip_1hr_in"].(string), 32)
	if err != nil {
		return weather, err
	}
	weather.Precip1hr = float32(precip1hr)
	precipToday, err := strconv.ParseFloat(data["precip_today_in"].(string), 32)
	if err != nil {
		return weather, err
	}
	weather.PrecipToday = float32(precipToday)
	visibility, err := strconv.ParseFloat(data["visibility_km"].(string), 32)
	weather.Visibility = float32(visibility)
	weather.WindDir = data["wind_dir"].(string)
	weather.WindSpeed = float32(data["wind_mph"].(float64))

	return weather, err
}

func (w *Weather) Save(c context.Context, timestamp int64) error {
	key := datastore.NewKey(c, "weather", "", timestamp, nil)
	key, err := datastore.Put(c, key, w)
	return err
}

func GetCurrentWeather(c context.Context) (*Weather, error) {
	key := datastore.NewKey(c, "weather", "current", 0, nil)
	var weather = new(Weather)
	err := nds.Get(c, key, weather)
	return weather, err
}

func GetCurrentStatus(c context.Context) (*prt.Status, error) {
	key := datastore.NewKey(c, "updates", "current", 0, nil)
	var status = new(prt.Status)
	err := nds.Get(c, key, status)
	return status, err
}

func StoreNewStatus(c context.Context, status *prt.Status) error {
	curStatus := datastore.NewKey(c, "updates", "current", 0, nil)
	curStatus, err := nds.Put(c, curStatus, status)
	if err != nil {
		return err
	}

	statusKey := datastore.NewKey(c, "updates", "", status.Timestamp, nil)
	statusKey, err = datastore.Put(c, statusKey, status)

	return err
}

func NotifyFCM(c context.Context, status *prt.Status, weather *Weather) (fcm.Response, error) {
	var (
		cfg, err    = config.Load(c)
		urlClient   = urlfetch.Client(c)
		fcmClient   = fcm.NewFCMWithClient(cfg["fcmKey"], urlClient)
		weatherData map[string]interface{}
		data        map[string]string
	)
	if err != nil {
		return fcm.Response{}, err
	}
	weatherBytes, err := json.Marshal(weather)
	if err != nil {
		return fcm.Response{}, err
	}
	json.Unmarshal(weatherBytes, &weatherData)

	data = status.ToMap()
	log.Infof(c, "Adding %s %s %#v to data %#v", weather.Conditions, string(weatherBytes[:]), weatherData, data)
	for k, v := range weatherData {
		data[k] = fmt.Sprint(v)
	}
	log.Infof(c, "Data after add: %#v", data)

	return fcmClient.Send(fcm.Message{
		Data: data,
		To:   FCMTopic,
	})
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
