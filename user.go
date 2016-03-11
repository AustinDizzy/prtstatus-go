package main

import (
	"github.com/austindizzy/prtstatus-go/prt"
	"time"
)

type User struct {
	Key              string
	Device           string
	RegistrationDate time.Time
}

func NewUser(opts ...string) *User {
	u := &User{}
	if len(opts) >= 1 {
		u.Key = opts[0]
	}
	if len(opts) >= 2 {
		u.Device = opts[0]
	}
	return u
}

var userColumns = []string{"key", "device", "registration_date"}

func storeUser(user *User) error {
	if user.RegistrationDate.IsZero() {
		user.RegistrationDate = time.Now()
	}

	_, err := DB.ExecOne(`
        INSERT INTO users (key, device, registration_date)
        VALUES (?, ?, ?)
    `, user.Key, user.Device, user.RegistrationDate)

	return err
}

func getUsers(device ...string) ([]User, error) {
	var (
		err   error
		users []User
		where = ``
		param interface{}
	)

	if len(device) == 1 {
		where = `WHERE device = ?`
		param = device[0]
	} else if len(device) > 1 {
		where = `WHERE device IN ?`
		param = device
	}

	_, err = DB.Query(&users, `
        SELECT key, device, registration_date
        FROM users
        `+where, param)

	return users, err
}

func getUserIDs(users []User) []string {
	s := []string{}
	for _, u := range users {
		s = append(s, u.Key)
	}
	return s
}

func (u User) send(data *prt.Status) error {
	return sendToUser(data, u)
}

func (u *User) delete() error {
	return deleteUser(u.Key)
}

func deleteUser(id string) (err error) {
	_, err = DB.Exec(`
        DELETE FROM users
        WHERE key = ?
        LIMIT 1
    `, id)

	return err
}

func userCount() (n int) {
	_, err := DB.Query(&n, `SELECT count(*)::int FROM users`)
	if err != nil {
		return -1
	}
	return
}
