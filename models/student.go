package models

import "github.com/uadmin/uadmin"

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

func (s Student) String() string {
	return s.FirstName + " " + s.LastName
}
