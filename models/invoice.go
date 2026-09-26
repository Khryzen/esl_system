package models

import (
	"fmt"
	"time"

	"github.com/uadmin/uadmin"
)

type Invoice struct {
	uadmin.Model
	InvoiceNumber string
	TransactionID string
	InvoiceDate   time.Time
	DueDate       time.Time
	PaidDate      *time.Time
	Student       Student
	StudentID     uint
	Enrollment    Enrollment
	EnrollmentID  uint
	Amount        float64 `uadmin:"required"`
	Paid          bool    `uadmin:"required"`
}

// Save gives a new invoice its number, then saves it. Unlike Enrollment's random
// ReferenceNumber (an internal lookup key), an invoice number is something a student
// might actually see on a document, so this uses a plain sequential format instead:
// INV-000001, INV-000002, and so on. That needs the row's own auto-incrementing ID,
// which only exists after the first save — hence the two-step save below. Later
// saves (paying an invoice, etc.) never touch an InvoiceNumber that's already set.
func (i *Invoice) Save() {
	isNew := i.ID == 0
	uadmin.Save(i)

	if isNew && i.InvoiceNumber == "" {
		i.InvoiceNumber = fmt.Sprintf("INV-%06d", i.ID)
		uadmin.Save(i)
	}
}
