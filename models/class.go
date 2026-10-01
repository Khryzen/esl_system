package models

import (
	"errors"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrClassEnrollmentRequired = errors.New(
		"Enrollment is required.",
	)

	ErrClassEnrollmentNotFound = errors.New(
		"The selected enrollment could not be found.",
	)

	ErrClassEnrollmentInactive = errors.New(
		"The selected enrollment is no longer active.",
	)

	ErrClassNoCreditsRemaining = errors.New(
		"The selected enrollment has no classes remaining.",
	)

	ErrClassScheduleFailed = errors.New(
		"The class could not be scheduled. Please try again.",
	)
)

type Class struct {
	uadmin.Model
	ClassDate time.Time
	StartTime *time.Time
	EndTime   *time.Time

	Enrollment   Enrollment
	EnrollmentID uint

	Course   Course
	CourseID uint

	Student   Student
	StudentID uint

	Present        bool
	Absent         bool
	CreditRefunded bool
}

func (c *Class) Schedule() error {
	if c.EnrollmentID == 0 {
		return ErrClassEnrollmentRequired
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var enrollment Enrollment

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&enrollment, c.EnrollmentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassEnrollmentNotFound
			}

			return err
		}

		if !enrollment.Active {
			return ErrClassEnrollmentInactive
		}

		if enrollment.ClassesRemaining <= 0 {
			return ErrClassNoCreditsRemaining
		}

		c.StudentID = enrollment.StudentID
		c.CourseID = enrollment.CourseID

		if err := tx.Create(c).Error; err != nil {
			return err
		}

		enrollment.ClassesRemaining--

		if enrollment.ClassesRemaining == 0 {
			enrollment.Active = false
		}

		if err := tx.Save(&enrollment).Error; err != nil {
			return err
		}

		c.Enrollment = enrollment

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrClassEnrollmentRequired),
		errors.Is(err, ErrClassEnrollmentNotFound),
		errors.Is(err, ErrClassEnrollmentInactive),
		errors.Is(err, ErrClassNoCreditsRemaining):
		return err
	}

	return errors.Join(ErrClassScheduleFailed, err)
}
