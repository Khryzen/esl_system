package models

import "github.com/uadmin/uadmin"

type Teacher struct {
	uadmin.Model

	FirstName string `uadmin:"required"`
	LastName  string `uadmin:"required"`

	User   uadmin.User
	UserID uint

	// Contact information
	Email string
	Phone string

	// Professional profile
	ProfilePhoto    string
	Bio             string `uadmin:"html"`
	Languages       string
	Specializations string
	Education       string
	YearsExperience int
}
