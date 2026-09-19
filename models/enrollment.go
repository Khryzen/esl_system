package models

import (
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/uadmin/uadmin"
)

const (
	enrollmentRefLength   = 12
	enrollmentRefChars    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	enrollmentRefAttempts = 10
)

type Enrollment struct {
	uadmin.Model
	ReferenceNumber  string
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

func (e *Enrollment) Save() {
	if e.ReferenceNumber == "" {
		ref, err := newEnrollmentRef()
		if err != nil {
			uadmin.Trail(uadmin.ERROR, "Enrollment: could not create a reference number: %v", err)
			return
		}
		e.ReferenceNumber = ref
	}
	uadmin.Save(e)
}

func newEnrollmentRef() (string, error) {
	for i := 0; i < enrollmentRefAttempts; i++ {
		ref, err := randomEnrollmentRef()
		if err != nil {
			return "", err
		}

		taken := []Enrollment{}
		uadmin.Filter(&taken, "reference_number = ?", ref)
		if len(taken) == 0 {
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
