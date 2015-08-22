package main

import (
	"time"
)

func storeData(d *PRTStatus) error {
	r, err := DB.QueryOne(d, `
		INSERT INTO updates (status, message, timestamp, stations, busses_dispatched)
		VALUES (?::integer, ?::text, ?::bigint, ?, ?::boolean)
		RETURNING id
	`, d.Status, d.Message, d.Timestamp, d.getStations(), d.bussesRunning())
	if r != nil && r.Affected() == 0 {
		LogErr(err, "storing data", &d)
	} else {
		LogErr(err)
	}
	return err
}

func getLastData() (PRTStatus, error) {
	lastStatus := PRTStatus{}
	_, err := DB.QueryOne(&lastStatus, `
		SELECT id, status, message, timestamp, stations, busses_dispatched
		FROM updates
		ORDER BY id DESC LIMIT 1
	`)
	LogErr(err, "getting last update", lastStatus)
	return lastStatus, err
}

func getData(n time.Duration) ([]PRTStatus, error) {
	var updates Updates
	now := time.Now()
	begin := now.Add(-n)

	_, err := DB.Query(&updates, `
		SELECT * FROM updates WHERE timestamp BETWEEN ?::bigint AND ?::bigint
	`, begin.Format("1136239445"), now.Format("1136239445"))

	LogErr(err, "getting data from", begin, "to", now)
	return updates.C, err
}

func (users *Users) NewRecord() interface{} {
	users.C = append(users.C, User{})
	return &users.C[len(users.C)-1]
}

func (p *PRTStatus) getStations() []string {
	s := []string{}
	for i := range p.stationsData {
		s = append(s, p.stationsData[i].Name)
	}
	return s
}

func (p *PRTStatus) bussesRunning() bool {
	return (p.bussesDispatchedStr != "0")
}

func (a *PRTStatus) compare(b PRTStatus) bool {
	return (a.Status == b.Status &&
		a.Message == b.Message &&
		a.Timestamp == b.Timestamp)
}

func (updates *Updates) NewRecord() interface{} {
	updates.C = append(updates.C, PRTStatus{})
	return &updates.C[len(updates.C)-1]
}
