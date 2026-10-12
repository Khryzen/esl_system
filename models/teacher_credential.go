package models

import (
	"errors"
	"strings"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

var (
	ErrCredentialTitleRequired   = errors.New("credential title is required")
	ErrCredentialTeacherRequired = errors.New("credential teacher is required")
	ErrCredentialTeacherNotFound = errors.New("credential teacher was not found")
	ErrCredentialInvalidDates    = errors.New("expiration date cannot be before issue date")
)

type TeacherCredential struct {
	uadmin.Model
	Teacher             Teacher
	TeacherID           uint
	Title               string `uadmin:"required"`
	IssuingOrganization string
	CredentialType      string
	Description         string
	IssueDate           *time.Time
	ExpirationDate      *time.Time
	DocumentPath        string
	IsPublic            bool
}

// BeforeSave validates the credential and protects its teacher association.
func (c *TeacherCredential) BeforeSave(tx *gorm.DB) error {
	if err := validateTeacherCredential(c); err != nil {
		return err
	}

	// Confirm the credential references an existing teacher.
	var teacher Teacher
	if err := tx.First(&teacher, c.TeacherID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCredentialTeacherNotFound
		}
		return err
	}

	if c.ID == 0 {
		return nil
	}

	var existing TeacherCredential
	if err := tx.Unscoped().First(&existing, c.ID).Error; err != nil {
		return err
	}

	// A credential cannot be reassigned through an ordinary update.
	c.TeacherID = existing.TeacherID

	return nil
}

func validateTeacherCredential(c *TeacherCredential) error {
	if strings.TrimSpace(c.Title) == "" {
		return ErrCredentialTitleRequired
	}

	if c.TeacherID == 0 {
		return ErrCredentialTeacherRequired
	}

	if c.IssueDate != nil &&
		c.ExpirationDate != nil &&
		c.ExpirationDate.Before(*c.IssueDate) {
		return ErrCredentialInvalidDates
	}

	return nil
}
