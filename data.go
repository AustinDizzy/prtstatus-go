package main

import (
	"github.com/austindizzy/prtstatus-go/prt"
	"gopkg.in/pg.v4"
)

func initDatabase(user, database string) {
	DB = pg.Connect(&pg.Options{
		User:     user,
		Database: database,
	})
}

var statusColumns = []string{"status", "message", "timestamp", "stations", "busses_dispatched", "duration"}

func saveStatus(d *prt.Status) error {
	_, err := DB.Exec(`
        INSERT INTO updates (status, message, timestamp, stations, busses_dispatched, duration)
        VALUES (?, ?, ?, ?, ?, ?)
    `, d.Status, d.Message, d.Timestamp, d.Stations, d.BussesDispatched, d.Duration)
	return err
}

func getLastStatus() (prt.Status, error) {
	var (
		lastStatus prt.Status
		err        error
		q          = `
            SELECT status, message, timestamp, stations, busses_dispatched, duration
            FROM updates 
            ORDER BY id DESC 
            LIMIT 1
        `
	)

	_, err = DB.QueryOne(&lastStatus, q)
	if err == pg.ErrNoRows {
		err = nil
	}
	return lastStatus, err
}
