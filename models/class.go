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

	ErrClassAlreadyCancelled = errors.New(
		"The class has already been cancelled.",
	)

	ErrClassNotCancelled = errors.New(
		"The class must be cancelled before its credit can be refunded.",
	)

	ErrClassCreditAlreadyRefunded = errors.New(
		"The class credit has already been refunded.",
	)

	ErrClassCancelFailed = errors.New(
		"The class could not be cancelled. Please try again.",
	)

	ErrClassCreditRefundFailed = errors.New(
		"The class credit could not be refunded. Please try again.",
	)

	ErrClassNotFound = errors.New(
		"The selected class could not be found.",
	)

	ErrClassRescheduleFailed = errors.New(
		"The class could not be rescheduled. Please try again.",
	)

	ErrClassRescheduleCourseChanged = errors.New(
		"The class cannot be rescheduled because the enrollment course has changed.",
	)

	ErrClassAttendanceRecorded = errors.New(
		"The class attendance has already been recorded.",
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
	Cancelled      bool
	CreditRefunded bool

	RescheduledFromID uint
	RescheduledFrom   *Class
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

func (c *Class) Cancel() error {
	if c.ID == 0 {
		return ErrClassNotFound
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var class Class

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&class, c.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassNotFound
			}

			return err
		}

		if class.Cancelled {
			return ErrClassAlreadyCancelled
		}

		if class.Present || class.Absent {
			return ErrClassAttendanceRecorded
		}

		class.Cancelled = true

		if err := tx.Save(&class).Error; err != nil {
			return err
		}

		c.Cancelled = class.Cancelled

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrClassNotFound),
		errors.Is(err, ErrClassAlreadyCancelled),
		errors.Is(err, ErrClassAttendanceRecorded):
		return err
	}

	return errors.Join(ErrClassCancelFailed, err)
}

func (c *Class) RefundCredit() error {
	if c.ID == 0 {
		return ErrClassNotFound
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var class Class

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&class, c.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassNotFound
			}

			return err
		}

		if !class.Cancelled {
			return ErrClassNotCancelled
		}

		if class.CreditRefunded {
			return ErrClassCreditAlreadyRefunded
		}

		if class.EnrollmentID == 0 {
			return ErrClassEnrollmentNotFound
		}

		var enrollment Enrollment

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&enrollment, class.EnrollmentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassEnrollmentNotFound
			}

			return err
		}

		enrollment.ClassesRemaining++

		if enrollment.ClassesRemaining > 0 {
			enrollment.Active = true
		}

		class.CreditRefunded = true

		if err := tx.Save(&enrollment).Error; err != nil {
			return err
		}

		if err := tx.Save(&class).Error; err != nil {
			return err
		}

		c.CreditRefunded = class.CreditRefunded

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrClassNotFound),
		errors.Is(err, ErrClassNotCancelled),
		errors.Is(err, ErrClassCreditAlreadyRefunded),
		errors.Is(err, ErrClassEnrollmentNotFound):
		return err
	}

	return errors.Join(ErrClassCreditRefundFailed, err)
}

func (c *Class) Reschedule(replacement *Class) error {
	if c.ID == 0 {
		return ErrClassNotFound
	}

	if replacement == nil {
		return ErrClassRescheduleFailed
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var original Class

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&original, c.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassNotFound
			}

			return err
		}

		if original.Cancelled {
			return ErrClassAlreadyCancelled
		}

		if original.EnrollmentID == 0 {
			return ErrClassEnrollmentNotFound
		}

		var enrollment Enrollment

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&enrollment, original.EnrollmentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassEnrollmentNotFound
			}

			return err
		}

		if original.CourseID != enrollment.CourseID {
			return ErrClassRescheduleCourseChanged
		}

		// Refund the original class credit.
		// The refunded credit will immediately be consumed
		// by the replacement class, so the enrollment's
		// final credit balance remains unchanged.
		enrollment.ClassesRemaining++

		if enrollment.ClassesRemaining > 0 {
			enrollment.Active = true
		}

		original.Cancelled = true
		original.CreditRefunded = true

		// The replacement inherits the original enrollment,
		// student, and course.
		// Only schedule-related fields are supplied by the caller.
		replacement.EnrollmentID = original.EnrollmentID
		replacement.StudentID = enrollment.StudentID
		replacement.CourseID = enrollment.CourseID
		replacement.RescheduledFromID = original.ID
		replacement.Cancelled = false
		replacement.CreditRefunded = false

		if err := tx.Save(&original).Error; err != nil {
			return err
		}

		if err := tx.Save(&enrollment).Error; err != nil {
			return err
		}

		if err := tx.Create(replacement).Error; err != nil {
			return err
		}

		// The replacement consumes the credit that was
		// just refunded.
		enrollment.ClassesRemaining--

		if enrollment.ClassesRemaining == 0 {
			enrollment.Active = false
		}

		if err := tx.Save(&enrollment).Error; err != nil {
			return err
		}

		c.Cancelled = original.Cancelled
		c.CreditRefunded = original.CreditRefunded

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrClassNotFound),
		errors.Is(err, ErrClassAlreadyCancelled),
		errors.Is(err, ErrClassEnrollmentNotFound),
		errors.Is(err, ErrClassRescheduleCourseChanged):
		return err
	}

	return errors.Join(ErrClassRescheduleFailed, err)
}
