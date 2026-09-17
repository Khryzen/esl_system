package models

import (
	"strings"

	"github.com/uadmin/uadmin"
)

type Student struct {
	uadmin.Model
	FirstName string `uadmin:"required"`
	LastName  string
	WeChatID  string `uadmin:"required"`
	Email     string

	// This is for the user access if the client also wants to have a user account for the student
	User   uadmin.User
	UserID uint
}

func (s *Student) String() string {
	return s.FirstName + " " + s.LastName
}

func (s *Student) Save() map[string]any {
	user := uadmin.User{}
	user.FirstName = s.FirstName
	user.LastName = s.LastName
	username := strings.ReplaceAll(s.FirstName, " ", "")[:1] + strings.ReplaceAll(s.LastName, " ", "")
	user.Username = username
	user.Password = strings.ReplaceAll(s.FirstName, " ", "")[:1] + strings.ReplaceAll(s.LastName, " ", "")
	user.Active = true
	user.RemoteAccess = true

	user.Save()

	uadmin.Get(&user, "username = ?", username)
	s.UserID = user.ID
	uadmin.Save(s)

	return map[string]any{
		"username": user.Username,
		"password": user.Username,
	}
}
