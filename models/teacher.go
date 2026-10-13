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
	ErrTeacherNotFound          = errors.New("teacher profile was not found")
	ErrTeacherProfileIDRequired = errors.New("teacher profile ID is required")
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

func GetTeacherProfile() (*Teacher, error) {
	db := uadmin.GetDB()

	var teacher Teacher
	if err := db.First(&teacher).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}

		return nil, err
	}

	return &teacher, nil
}

// Update updates the editable fields of an existing teacher profile.
// The profile ID and linked user association cannot be changed.
func (t *Teacher) Update() error {
	if t.ID == 0 {
		return ErrTeacherProfileIDRequired
	}

	if err := validateTeacher(t); err != nil {
		return err
	}

	db := uadmin.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		var existing Teacher

		if err := tx.First(&existing, t.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeacherNotFound
			}
			return err
		}

		// Update only fields that are editable in the teacher profile.
		existing.FirstName = strings.TrimSpace(t.FirstName)
		existing.LastName = strings.TrimSpace(t.LastName)
		existing.Email = strings.TrimSpace(t.Email)
		existing.Phone = strings.TrimSpace(t.Phone)
		existing.ProfilePhoto = t.ProfilePhoto
		existing.Bio = t.Bio
		existing.Languages = t.Languages
		existing.Specializations = t.Specializations
		existing.Education = t.Education
		existing.YearsExperience = t.YearsExperience

		// UserID is intentionally not copied from the incoming profile.
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}

		// Reflect the saved record back into the caller's instance.
		*t = existing

		return nil
	})
}

func (t Teacher) String() string {
	return strings.TrimSpace(t.FirstName + " " + t.LastName)
}
