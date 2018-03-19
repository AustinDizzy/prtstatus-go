package user

import "time"

// User is the user receiving a notification
type User struct {
	Token            string `datastore:""`
	Device           string
	RegistrationDate time.Time
}

// NewUser creates a new User, where the options can be none or a combination of:
// one token string and one device type string
func NewUser(opts ...string) *User {
	var u = new(User)
	if len(opts) >= 1 {
		u.Token = opts[0]
	}
	if len(opts) >= 2 {
		u.Device = opts[1]
	}
	return u
}
