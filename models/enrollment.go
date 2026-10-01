package models

import (
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

const (
	enrollmentRefLength   = 12
	enrollmentRefChars    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	enrollmentRefAttempts = 10
)

var (
	ErrEnrollmentStudentRequired = errors.New("Student is required.")
	ErrEnrollmentCourseRequired  = errors.New("Course is required.")
	ErrEnrollmentPackageRequired = errors.New("Package is required.")

	ErrEnrollmentStudentNotFound = errors.New(
		"The selected student could not be found.",
	)
	ErrEnrollmentCourseNotFound = errors.New(
		"The selected course could not be found.",
	)
	ErrEnrollmentPackageNotFound = errors.New(
		"The selected package could not be found.",
	)

	ErrEnrollmentPackageNoClasses = errors.New(
		"The selected package has no available classes.",
	)

	ErrEnrollmentReferenceExists = errors.New(
		"The enrollment reference number is already in use.",
	)

	ErrEnrollmentReferenceGeneration = errors.New(
		"Could not generate an enrollment reference number. Please try again.",
	)

	ErrEnrollmentCreateFailed = errors.New(
		"The enrollment could not be created. Please try again.",
	)
)

type Enrollment struct {
	uadmin.Model
	ReferenceNumber  string `gorm:"uniqueIndex"`
	Student          Student
	StudentID        uint
	Course           Course
	CourseID         uint
	Package          Package
	PackageID        uint
	TotalClasses     int
	ClassesRemaining int
	Contract         string
	Active           bool
}

func (e *Enrollment) Create() error {
	if e.StudentID == 0 {
		return ErrEnrollmentStudentRequired
	}

	if e.CourseID == 0 {
		return ErrEnrollmentCourseRequired
	}

	if e.PackageID == 0 {
		return ErrEnrollmentPackageRequired
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var student Student

		if err := tx.First(&student, e.StudentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEnrollmentStudentNotFound
			}

			return err
		}

		var course Course

		if err := tx.First(&course, e.CourseID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEnrollmentCourseNotFound
			}

			return err
		}

		var pkg Package

		if err := tx.First(&pkg, e.PackageID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEnrollmentPackageNotFound
			}

			return err
		}

		if pkg.TotalClasses <= 0 {
			return ErrEnrollmentPackageNoClasses
		}

		if e.ReferenceNumber == "" {
			ref, err := newEnrollmentRefWithDB(tx)
			if err != nil {
				return ErrEnrollmentReferenceGeneration
			}

			e.ReferenceNumber = ref
		} else {
			var count int64

			if err := tx.Model(&Enrollment{}).
				Where("reference_number = ?", e.ReferenceNumber).
				Count(&count).Error; err != nil {
				return err
			}

			if count > 0 {
				return ErrEnrollmentReferenceExists
			}
		}

		e.Student = student
		e.Course = course
		e.Package = pkg
		e.TotalClasses = pkg.TotalClasses
		e.ClassesRemaining = pkg.TotalClasses

		if err := tx.Create(e).Error; err != nil {
			return err
		}

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrEnrollmentStudentRequired),
		errors.Is(err, ErrEnrollmentCourseRequired),
		errors.Is(err, ErrEnrollmentPackageRequired),
		errors.Is(err, ErrEnrollmentStudentNotFound),
		errors.Is(err, ErrEnrollmentCourseNotFound),
		errors.Is(err, ErrEnrollmentPackageNotFound),
		errors.Is(err, ErrEnrollmentPackageNoClasses),
		errors.Is(err, ErrEnrollmentReferenceExists),
		errors.Is(err, ErrEnrollmentReferenceGeneration):
		return err
	}

	// Preserve the underlying database error for logging/debugging while
	// giving callers a stable top-level business error.
	return errors.Join(ErrEnrollmentCreateFailed, err)
}

func (e *Enrollment) Save() {
	if e.ReferenceNumber == "" {
		ref, err := newEnrollmentRef()
		if err != nil {
			uadmin.Trail(
				uadmin.ERROR,
				"Enrollment: could not create a reference number: %v",
				err,
			)
			return
		}

		e.ReferenceNumber = ref
	}

	uadmin.Save(e)
}

func newEnrollmentRef() (string, error) {
	return newEnrollmentRefWithDB(uadmin.GetDB())
}

func newEnrollmentRefWithDB(db *gorm.DB) (string, error) {
	for i := 0; i < enrollmentRefAttempts; i++ {
		ref, err := randomEnrollmentRef()
		if err != nil {
			return "", err
		}

		var count int64

		if err := db.Model(&Enrollment{}).
			Where("reference_number = ?", ref).
			Count(&count).Error; err != nil {
			return "", err
		}

		if count == 0 {
			return ref, nil
		}
	}

	return "", errors.New("could not find an unused reference number")
}

func randomEnrollmentRef() (string, error) {
	charCount := big.NewInt(int64(len(enrollmentRefChars)))
	ref := make([]byte, enrollmentRefLength)

	for i := range ref {
		n, err := rand.Int(rand.Reader, charCount)
		if err != nil {
			return "", err
		}

		ref[i] = enrollmentRefChars[n.Int64()]
	}

	return string(ref), nil
}
