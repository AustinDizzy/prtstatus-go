package main

import (
	"github.com/austindizzy/prtstatus-go/prt"
	"github.com/spf13/viper"
)

func poll() {
	currentStatus, err := prt.GetStatus()
	log(err, currentStatus)

	lastStatus, err := getLastStatus()
	log(err, lastStatus)

	if currentStatus.CompareTo(lastStatus) != 0 {
		log(saveStatus(currentStatus), "PRT Update")

		if viper.GetBool("live") == true {
			err = pushToUsers(currentStatus)
			log(err)
		} else {
			log(nil, "Alert Users", currentStatus)
		}
	}
}

func pushToUsers(data *prt.Status) error {
	users, err := getUsers()
	if err != nil {
		return err
	}

	p := NewPush(data, users...)

	err = p.ToAndroid()
	if err != nil {
		return err
	}

	err = p.ToPushbullet()
	return err
}
