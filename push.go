package main

import (
	"fmt"
	"github.com/alexjlockwood/gcm"
	"github.com/austindizzy/prtstatus-go/prt"
	"github.com/fatih/structs"
	"github.com/spf13/viper"
	"github.com/xconstruct/go-pushbullet"
)

type pushRequest struct {
	data  *prt.Status
	users []User
}

//NewPush returns  a new push request that is used to deliver
//data to users. This is used in both batch and single-case posting.
func NewPush(data *prt.Status, user ...User) *pushRequest {
	return &pushRequest{
		data:  data,
		users: user,
	}
}

func (p pushRequest) ToAndroid(ids ...string) error {
	var (
		err      error
		androids []string
	)

	if len(p.users) != 0 {
		androids = getUserIDs(p.users, "android")
	}

	if len(ids) != 0 && len(androids) == 0 {
		androids = ids
	}

	if len(androids) > 0 {
		var (
			data      = structs.Map(p.data)
			payload   = gcm.NewMessage(data, androids...)
			sender    = gcm.Sender{ApiKey: viper.GetString("gcmKey")}
			resp, err = sender.Send(payload, 5)
		)

		for i, result := range resp.Results {
			if result.Error == "NotRegistered" {
				log(deleteUser(androids[i]))
			} else if len(result.RegistrationID) != 0 {
				_, err = DB.ExecOne(`
                UPDATE users
                SET key = ?
                WHERE key = ?, device = 'android'
            `, result.RegistrationID, androids[i])
				log(err, "user updated: ", androids[i], result.RegistrationID)
			}
		}
	}

	return err
}

func (p pushRequest) ToPushbullet() error {
	var (
		err  error
		data = p.getPBPayload("channel_tag", "wvuprtstatus")
		pb   = pushbullet.New(viper.GetString("pushbullet.api_key"))
	)

	err = pb.Push("/pushes", data)
	if err != nil {
		return err
	}

	return nil
}

func (p pushRequest) getPBPayload(target, key string) map[string]string {
	var (
		status string
		data   = map[string]string{
			"title": "The PRT is %s.",
			"body":  p.data.Message,
			"type":  "note",
			target:  key,
		}
	)

	if p.data.Status == 1 {
		status = "UP"
	} else if p.data.Status == 9 {
		status = "CLOSED"
	} else {
		status = "DOWN"
	}

	data["title"] = fmt.Sprintf(data["title"], status)

	return data
}
