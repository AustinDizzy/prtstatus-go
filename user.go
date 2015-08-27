package main

import (
	"github.com/alexjlockwood/gcm"
	"github.com/fatih/structs"
	"time"
)

func storeUser(user *User) error {
	if user.RegistrationDate.IsZero() {
		user.RegistrationDate = time.Now()
	}

	_, err := DB.QueryOne(&user, `
		INSERT INTO users (registration_id, device, registration_date)
		VALUES (?::text, ?::text, ?::timestamp with time zone)
	`, user.RegistrationId, user.Device, user.RegistrationDate)
	LogErr(err, "inserting user")

	return err
}

func getUsers(device string) ([]User, error) {
	var users Users
	r, err := DB.Query(&users, `SELECT * FROM users`)
	LogErr(err, users.C, "getting all "+device+" users", r)
	return users.C, err
}

func getUserIDs(users []User) []string {
	s := []string{}
	for _, u := range users {
		s = append(s, u.RegistrationId)
	}
	return s
}

func sendToUser(data *PRTStatus, user ...User) error {
	ids := getUserIDs(user)
	LogErr(nil, "sending to IDs:", ids, structs.Map(data))
	payload := gcm.NewMessage(structs.Map(data), ids...)
	sender := &gcm.Sender{ApiKey: config.GCMKey}
	resp, err := sender.Send(payload, 5)

	for i, result := range resp.Results {
		if result.Error == "NotRegistered" {
			deleteUser(user[i].RegistrationId)
		}
		if len(result.RegistrationID) != 0 {
			_, e := DB.Exec(`
				UPDATE users SET registration_id = ?
				WHERE registration_id = ?
			`, result.RegistrationID, user[i].RegistrationId)
			LogErr(e, "user updated: ", user[i].RegistrationId, result.RegistrationID)
		}
	}

	LogErr(nil, resp.Success, "successful pushes")

	return err
}

func deleteUser(id string) error {
	_, err := DB.Exec(`DELETE FROM users WHERE registration_id = ?`, id)
	LogErr(err, "user deleted:", id)
	return err
}

func userCount() int {
	r, err := DB.Exec(`SELECT * FROM users`)
	LogErr(err, "counting users")
	return r.Affected()
}
