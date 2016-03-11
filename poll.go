package main

import (
	"github.com/alexjlockwood/gcm"
	"github.com/austindizzy/prtstatus-go/prt"
	"github.com/fatih/structs"
	"github.com/spf13/viper"
)

func poll() {
	currentStatus, err := prt.GetStatus()
	log(err, currentStatus)

	lastStatus, err := getLastStatus()
	log(err, lastStatus)

	if currentStatus.CompareTo(lastStatus) != 0 {
		log(saveStatus(currentStatus), "PRT Update")

		users, err := getUsers()
		log(err)

		if viper.GetBool("live") == true {
			err = sendToUser(currentStatus, users...)
			log(err)
		} else {
			log(nil, "Alert Users", currentStatus, getUserIDs(users))
		}
	}
}

func sendToUser(data *prt.Status, user ...User) error {
	var (
		ids       = getUserIDs(user)
		payload   = gcm.NewMessage(structs.Map(data), ids...)
		sender    = gcm.Sender{ApiKey: viper.GetString("gcmKey")}
		resp, err = sender.Send(payload, 5)
	)

	for i, result := range resp.Results {
		if result.Error == "NotRegistered" {
			log(user[i].delete())
		} else if len(result.RegistrationID) != 0 {
			_, err = DB.ExecOne(`
                UPDATE users SET key = ? WHERE key = ?
            `, result.RegistrationID, user[i].Key)
			log(err, "user updated: ", user[i].Key, result.RegistrationID)
		}
	}

	return err
}
