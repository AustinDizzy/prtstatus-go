package prt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const STATUS_API = "https://prtstatus.wvu.edu/api/%s?format=json"

type Status struct {
	Status           int
	Message          string
	Timestamp        time.Time
	Stations         []string
	BussesDispatched bool `db:"busses_dispatched"`
	Duration         string
}

type statusWrapper struct {
	Status    string
	Message   string
	Timestamp string
	Stations  []struct {
		Id   int
		Name string
	}
	BussesDispatched string
	Duration         []struct {
		Id      int
		Message string
	}
}

func GetStatus() (*Status, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, _ := http.NewRequest("GET", getStatusURL(), nil)
	req.Header.Set("Cache-Control", "max-age=0")
	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var data statusWrapper
	if err = json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	timestamp, _ := strconv.ParseInt(data.Timestamp, 10, 64)

	status := &Status{
		Message: data.Message,
	}

	status.Status, _ = strconv.Atoi(data.Status)
	status.Timestamp = time.Unix(timestamp, 0)
	status.BussesDispatched, _ = strconv.ParseBool(data.BussesDispatched)

	status.Stations = make([]string, len(data.Stations))
	for i, s := range data.Stations {
		status.Stations[i] = s.Name
	}

	if len(data.Duration) > 0 {
		status.Duration = data.Duration[0].Message
	}

	return status, nil
}

func getStatusURL() string {
	t := strconv.FormatInt(time.Now().Unix(), 10)
	return fmt.Sprintf(STATUS_API, t)
}

func (a Status) CompareTo(b Status) int {
	equal := a.Status == b.Status && a.Timestamp.Equal(b.Timestamp) && a.Message == b.Message
	if equal {
		return 0
	} else if a.Timestamp.Before(b.Timestamp) {
		return -1
	} else {
		return 1
	}
}
