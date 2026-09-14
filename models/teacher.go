package models

import "github.com/uadmin/uadmin"

type Teacher struct {
	uadmin.Model
	FirstName string `uadmin:"required"`
	LastName  string `uadmin:"required"`

	User   uadmin.User
	UserID uint
}
