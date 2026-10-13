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

// GetCredentials retrieves all credentials belonging to a teacher.
// Credentials are ordered by ID in ascending order.
func GetTeacherCredentials(teacherID uint) ([]TeacherCredential, error) {
	if teacherID == 0 {
		return nil, ErrTeacherNotFound
	}

	db := uadmin.GetDB()

	var teacher Teacher
	if err := db.First(&teacher, teacherID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}

	var credentials []TeacherCredential
	if err := db.
		Where("teacher_id = ?", teacherID).
		Order("id ASC").
		Find(&credentials).Error; err != nil {
		return nil, err
	}

	return credentials, nil
}

// Create persists a new teacher credential.
func (c *TeacherCredential) Create() error {
	if c == nil {
		return ErrCredentialTitleRequired
	}

	if c.ID != 0 {
		return errors.New("cannot create a credential that already has an ID")
	}

	if err := validateTeacherCredential(c); err != nil {
		return err
	}

	db := uadmin.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return err
		}

		return nil
	})
}

// Update persists changes to an existing teacher credential.
// The credential's ID and TeacherID cannot be changed through this method.
func (c *TeacherCredential) Update() error {
	if c == nil {
		return ErrCredentialTitleRequired
	}

	if c.ID == 0 {
		return errors.New("credential ID is required for update")
	}

	if err := validateTeacherCredential(c); err != nil {
		return err
	}

	db := uadmin.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		var existing TeacherCredential

		if err := tx.First(&existing, c.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return gorm.ErrRecordNotFound
			}
			return err
		}

		existing.Title = strings.TrimSpace(c.Title)
		existing.IssuingOrganization = c.IssuingOrganization
		existing.CredentialType = c.CredentialType
		existing.Description = c.Description
		existing.IssueDate = c.IssueDate
		existing.ExpirationDate = c.ExpirationDate
		existing.DocumentPath = c.DocumentPath
		existing.IsPublic = c.IsPublic

		if err := tx.Save(&existing).Error; err != nil {
			return err
		}

		*c = existing
		return nil
	})
}

// GetPublicTeacherCredentials retrieves only public credentials belonging
// to an existing teacher, ordered by ID in ascending order.
func GetPublicTeacherCredentials(teacherID uint) ([]TeacherCredential, error) {
	if teacherID == 0 {
		return nil, ErrTeacherNotFound
	}

	db := uadmin.GetDB()

	var teacher Teacher
	if err := db.First(&teacher, teacherID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}

	var credentials []TeacherCredential
	if err := db.
		Where("teacher_id = ? AND is_public = ?", teacherID, true).
		Order("id ASC").
		Find(&credentials).Error; err != nil {
		return nil, err
	}

	return credentials, nil
}

// Delete removes an existing teacher credential.
func (c *TeacherCredential) Delete() error {
	if c == nil {
		return ErrCredentialTitleRequired
	}

	if c.ID == 0 {
		return errors.New("credential ID is required for deletion")
	}

	db := uadmin.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		var existing TeacherCredential

		if err := tx.First(&existing, c.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return gorm.ErrRecordNotFound
			}
			return err
		}

		return tx.Delete(&existing).Error
	})
}
