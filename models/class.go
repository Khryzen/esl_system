package models

import (
	"time"

	"github.com/uadmin/uadmin"
)

type Class struct {
	uadmin.Model
	ClassDate      time.Time
	StartTime      *time.Time
	EndTime        *time.Time
	Enrollment     Enrollment
	EnrollmentID   uint
	Student        Student
	StudentID      uint
	Present        bool // true when tagged present
	Absent         bool // true when tagged absent
	CreditRefunded bool // true when an absence refunded a class credit
}
