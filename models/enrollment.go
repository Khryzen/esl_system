package models

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	ErrEnrollmentCourseInactive = errors.New(
		"The selected course is not available for enrollment.",
	)
	ErrEnrollmentPackageInactive = errors.New(
		"The selected package is not available for enrollment.",
	)
	ErrEnrollmentPackageExpired = errors.New(
		"The selected package is outside its validity period.",
	)
	ErrEnrollmentPackageNoClasses = errors.New(
		"The selected package has no available classes.",
	)

	ErrEnrollmentAlreadyExists = errors.New(
		"The student already has an active enrollment for this course.",
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

	ErrEnrollmentChangeCourseRequired = errors.New(
		"Course is required.",
	)

	ErrEnrollmentChangeCourseSame = errors.New(
		"The enrollment is already assigned to this course.",
	)

	ErrEnrollmentChangeCourseInactive = errors.New(
		"The selected course is not available.",
	)

	ErrEnrollmentChangeCourseNoCredits = errors.New(
		"The enrollment has no classes remaining.",
	)

	ErrEnrollmentChangeCourseFailed = errors.New(
		"The enrollment course could not be changed. Please try again.",
	)

	ErrEnrollmentNotFound = errors.New(
		"Enrollment not found. Please try again.",
	)

	ErrEnrollmentDeactivateAlreadyInactive = errors.New(
		"Enrollment is already inactive.",
	)

	ErrEnrollmentDeactivateFailed = errors.New(
		"The enrollment could not be deactivated. Please try again.",
	)
)

type enrollmentInternalSaveContextKey struct{}

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

func (e *Enrollment) BeforeSave(tx *gorm.DB) error {
	if e.ID == 0 {
		return nil
	}

	if tx.Statement.Context.Value(enrollmentInternalSaveContextKey{}) == true {
		return nil
	}

	var existing Enrollment

	if err := tx.Unscoped().First(&existing, e.ID).Error; err != nil {
		return err
	}

	e.StudentID = existing.StudentID
	e.CourseID = existing.CourseID
	e.PackageID = existing.PackageID
	e.TotalClasses = existing.TotalClasses
	e.ClassesRemaining = existing.ClassesRemaining
	e.Active = existing.Active
	e.ReferenceNumber = existing.ReferenceNumber

	return nil
}

func withEnrollmentInternalSave(tx *gorm.DB) *gorm.DB {
	ctx := context.WithValue(
		tx.Statement.Context,
		enrollmentInternalSaveContextKey{},
		true,
	)

	return tx.WithContext(ctx)
}

func (e *Enrollment) Create() error {
	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		return e.CreateWithTx(tx)
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
		errors.Is(err, ErrEnrollmentCourseInactive),
		errors.Is(err, ErrEnrollmentPackageInactive),
		errors.Is(err, ErrEnrollmentPackageExpired),
		errors.Is(err, ErrEnrollmentPackageNoClasses),
		errors.Is(err, ErrEnrollmentAlreadyExists),
		errors.Is(err, ErrEnrollmentReferenceExists),
		errors.Is(err, ErrEnrollmentReferenceGeneration):
		return err
	}

	return errors.Join(ErrEnrollmentCreateFailed, err)
}

func (e *Enrollment) CreateWithTx(tx *gorm.DB) error {
	if e.StudentID == 0 {
		return ErrEnrollmentStudentRequired
	}

	if e.CourseID == 0 {
		return ErrEnrollmentCourseRequired
	}

	if e.PackageID == 0 {
		return ErrEnrollmentPackageRequired
	}

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

	if !course.Active {
		return ErrEnrollmentCourseInactive
	}

	var pkg Package

	if err := tx.First(&pkg, e.PackageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEnrollmentPackageNotFound
		}

		return err
	}

	if !pkg.Active {
		return ErrEnrollmentPackageInactive
	}

	if !packageWithinValidityPeriod(pkg, time.Now()) {
		return ErrEnrollmentPackageExpired
	}

	if pkg.TotalClasses <= 0 {
		return ErrEnrollmentPackageNoClasses
	}

	var count int64

	if err := tx.Model(&Enrollment{}).
		Where(
			"student_id = ? AND course_id = ? AND active = ?",
			student.ID,
			course.ID,
			true,
		).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return ErrEnrollmentAlreadyExists
	}

	if e.ReferenceNumber == "" {
		ref, err := newEnrollmentRefWithDB(tx)

		if err != nil {
			return ErrEnrollmentReferenceGeneration
		}

		e.ReferenceNumber = ref
	} else {
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
	e.Active = true

	if err := tx.Create(e).Error; err != nil {
		return err
	}

	return nil
}

func packageWithinValidityPeriod(pkg Package, now time.Time) bool {
	if pkg.ValidFrom == nil || pkg.ValidUntil == nil {
		return false
	}

	return !now.Before(*pkg.ValidFrom) &&
		!now.After(*pkg.ValidUntil)
}

func (e *Enrollment) ChangeCourse(courseID uint) error {
	if courseID == 0 {
		return ErrEnrollmentChangeCourseRequired
	}

	if e.ID == 0 {
		return ErrEnrollmentNotFound
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var enrollment Enrollment

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&enrollment, e.ID).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEnrollmentNotFound
			}

			return err
		}

		if !enrollment.Active {
			return ErrEnrollmentChangeCourseNoCredits
		}

		if enrollment.ClassesRemaining <= 0 {
			return ErrEnrollmentChangeCourseNoCredits
		}

		if enrollment.CourseID == courseID {
			return ErrEnrollmentChangeCourseSame
		}

		var course Course

		if err := tx.First(&course, courseID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEnrollmentCourseNotFound
			}

			return err
		}

		if !course.Active {
			return ErrEnrollmentChangeCourseInactive
		}

		var count int64

		if err := tx.Model(&Enrollment{}).
			Where(
				"student_id = ? AND course_id = ? AND active = ? AND id != ?",
				enrollment.StudentID,
				course.ID,
				true,
				enrollment.ID,
			).
			Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return ErrEnrollmentAlreadyExists
		}

		enrollment.CourseID = course.ID
		enrollment.Course = course

		if err := withEnrollmentInternalSave(tx).
			Save(&enrollment).Error; err != nil {
			return err
		}

		e.CourseID = enrollment.CourseID
		e.Course = course
		e.StudentID = enrollment.StudentID
		e.PackageID = enrollment.PackageID
		e.TotalClasses = enrollment.TotalClasses
		e.ClassesRemaining = enrollment.ClassesRemaining
		e.Active = enrollment.Active

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrEnrollmentChangeCourseRequired),
		errors.Is(err, ErrEnrollmentNotFound),
		errors.Is(err, ErrEnrollmentChangeCourseNoCredits),
		errors.Is(err, ErrEnrollmentChangeCourseSame),
		errors.Is(err, ErrEnrollmentCourseNotFound),
		errors.Is(err, ErrEnrollmentChangeCourseInactive),
		errors.Is(err, ErrEnrollmentAlreadyExists):
		return err
	}

	return errors.Join(ErrEnrollmentChangeCourseFailed, err)
}

func (e *Enrollment) Deactivate() error {
	if e.ID == 0 {
		return ErrEnrollmentNotFound
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var enrollment Enrollment

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&enrollment, e.ID).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEnrollmentNotFound
			}

			return err
		}

		if !enrollment.Active {
			return ErrEnrollmentDeactivateAlreadyInactive
		}

		enrollment.Active = false

		if err := withEnrollmentInternalSave(tx).
			Save(&enrollment).Error; err != nil {
			return err
		}

		e.StudentID = enrollment.StudentID
		e.CourseID = enrollment.CourseID
		e.PackageID = enrollment.PackageID
		e.TotalClasses = enrollment.TotalClasses
		e.ClassesRemaining = enrollment.ClassesRemaining
		e.Active = enrollment.Active
		e.ReferenceNumber = enrollment.ReferenceNumber

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrEnrollmentNotFound),
		errors.Is(err, ErrEnrollmentDeactivateAlreadyInactive):
		return err
	}

	return errors.Join(ErrEnrollmentDeactivateFailed, err)
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
