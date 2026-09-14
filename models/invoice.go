package models

import (
	"time"

	"github.com/uadmin/uadmin"
)

type Invoice struct {
	uadmin.Model
	TransactionID string
	InvoiceDate   time.Time
	Student       Student
	StudentID     uint
	Enrollment    Enrollment
	EnrollmentID  uint
	Amount        float64 `uadmin:"required"`
	Paid          bool    `uadmin:"required"`
}
