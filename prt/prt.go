package prt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// StatusAPI is the endpoint for the PRT status
const StatusAPI = "https://prtstatus.wvu.edu/api/%s?format=json"

// Status is a PRT status update
type Status struct {
	// Status can be an integer from 1 through 7 meaning the following:
	// 1 - normal, 2 - down between A and B, 3 - down for maintenance, 4 - down, 5 - special event, 6 - down at A, 7 - closed
	// However, PRT operators seem to play favorites with just 1, 5, and 7 even though others are more applicable
	Status int `json:"status"`
	// Message will contain the message text as entered by the PRT operators
	Message string `json:"message"`
	// Timestamp is a 64-bit integer of a Unix timestamp of the status event
	Timestamp int64 `json:"timestamp"`
	// Stations is a string array of the stations the event occurred at
	Stations []string `json:"stations"`
	// BussesDispatched is a boolean true if the event required busses to be dispatched
	BussesDispatched bool `json:"bussesDispatched"`
	// Duration is a so far unused string variable
	Duration string `json:"duration"`
}

type statusWrapper struct {
	Status    string
	Message   string
	Timestamp string
	Stations  []struct {
		ID   int
		Name string
	}
	BussesDispatched string
	Duration         []struct {
		ID      int
		Message string
	}
}

// GetStatusText returns a simple English representation of the PRT's status
func (status *Status) GetStatusText() string {
	if status.IsOpen() {
		return "UP"
	} else if status.IsClosed() {
		return "CLOSED"
	}
	return "DOWN"
}

// IsClosed returns if the PRT is closed
func (status *Status) IsClosed() bool {
	return status.Status == 7
}

// IsOpen returns if the PRT is open
func (status *Status) IsOpen() bool {
	return status.Status == 1
}

// IsDown returns if the PRT is not functioning
func (status *Status) IsDown() bool {
	return !status.IsOpen() && !status.IsClosed()
}

// ToMap returns a key-value map[string]string of the event
func (status *Status) ToMap() map[string]string {
	return map[string]string{
		"status":           fmt.Sprint(status.Status),
		"message":          fmt.Sprint(status.Message),
		"timestamp":        fmt.Sprint(status.Timestamp),
		"stations":         strings.Join(status.Stations, ","),
		"bussesDispatched": fmt.Sprint(status.BussesDispatched),
	}
}

func (status *Status) fromJSON(body []byte) error {
	var data statusWrapper
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	status.Message = data.Message
	status.Status, _ = strconv.Atoi(data.Status)
	timestamp, _ := strconv.ParseInt(data.Timestamp, 10, 64)
	status.Timestamp = timestamp
	status.BussesDispatched, _ = strconv.ParseBool(data.BussesDispatched)
	status.Stations = make([]string, len(data.Stations))
	for i, s := range data.Stations {
		status.Stations[i] = s.Name
	}

	if len(data.Duration) > 0 {
		status.Duration = data.Duration[0].Message
	}
	return nil
}

// StatusClient is the client used in go to intermediate status request
type StatusClient struct {
	c *http.Client
}

// NewClient returns a new StatusClient to use to query for PRT status updates
func NewClient(c ...*http.Client) *StatusClient {
	s := new(StatusClient)
	if len(c) > 0 {
		s.c = c[0]
	} else {
		s.c = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return s
}

// GetCurrentStatus retuns the current status from the live status API
func (c *StatusClient) GetCurrentStatus() (*Status, string, error) {
	return c.GetStatus(time.Now())
}

// GetStatus returns a status from the live status API relative to the supplied time
func (c *StatusClient) GetStatus(t time.Time) (*Status, string, error) {
	req, _ := http.NewRequest("GET", getStatusURL(t), nil)
	req.Header.Set("Cache-Control", "max-age=0")
	res, err := c.c.Do(req)

	if err != nil {
		return nil, "", err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, string(body[:]), err
	}

	status := new(Status)
	return status, string(body[:]), status.fromJSON(body)
}

func getStatusURL(t time.Time) string {
	return fmt.Sprintf(StatusAPI, strconv.FormatInt(t.Unix(), 10))
}

// CompareTo compares a Status update A with B, useful for sorting chronologically
func (status *Status) CompareTo(b *Status) int {
	equal := status.Status == b.Status && status.Timestamp == b.Timestamp && status.Message == b.Message
	if equal {
		return 0
	} else if status.Timestamp < b.Timestamp {
		return -1
	} else {
		return 1
	}
}
