package models

import (
	"errors"
	"strings"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

var (
	ErrTeacherFirstNameRequired = errors.New("teacher first name is required")
	ErrTeacherLastNameRequired  = errors.New("teacher last name is required")
	ErrTeacherInvalidExperience = errors.New("years of experience cannot be negative")
)

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

// BeforeSave validates profile fields and protects the linked user.
func (t *Teacher) BeforeSave(tx *gorm.DB) error {
	if err := validateTeacher(t); err != nil {
		return err
	}

	if t.ID == 0 {
		return nil
	}

	var existing Teacher
	if err := tx.Unscoped().First(&existing, t.ID).Error; err != nil {
		return err
	}
	t.UserID = existing.UserID
	return nil
}

func validateTeacher(t *Teacher) error {
	if strings.TrimSpace(t.FirstName) == "" {
		return ErrTeacherFirstNameRequired
	}

	if strings.TrimSpace(t.LastName) == "" {
		return ErrTeacherLastNameRequired
	}

	if t.YearsExperience < 0 {
		return ErrTeacherInvalidExperience
	}

	return nil
}

func (t Teacher) String() string {
	return strings.TrimSpace(t.FirstName + " " + t.LastName)
}
