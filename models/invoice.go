package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvoiceNotFound    = errors.New("invoice not found")
	ErrInvoiceAlreadyPaid = errors.New("invoice is already paid")
)

type invoiceInternalSaveContextKey struct{}

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

func (i *Invoice) BeforeSave(tx *gorm.DB) error {
	if tx.Statement.Context.Value(invoiceInternalSaveContextKey{}) == true {
		return nil
	}

	if i.ID == 0 {
		return nil
	}

	var existing Invoice

	if err := tx.Unscoped().First(&existing, i.ID).Error; err != nil {
		return err
	}

	i.InvoiceNumber = existing.InvoiceNumber
	i.TransactionID = existing.TransactionID
	i.InvoiceDate = existing.InvoiceDate
	i.DueDate = existing.DueDate
	i.PaidDate = existing.PaidDate
	i.StudentID = existing.StudentID
	i.EnrollmentID = existing.EnrollmentID
	i.Amount = existing.Amount
	i.Paid = existing.Paid

	return nil
}

func withInvoiceInternalSave(tx *gorm.DB) *gorm.DB {
	ctx := context.WithValue(
		tx.Statement.Context,
		invoiceInternalSaveContextKey{},
		true,
	)

	return tx.WithContext(ctx)
}

// Create persists a new invoice. If no invoice number was supplied, it generates
// one from the newly assigned database ID and persists that number in the same
// transaction.
func (i *Invoice) Create() error {
	if i.ID != 0 {
		return errors.New("invoice already exists")
	}

	db := uadmin.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(i).Error; err != nil {
			return err
		}

		if i.InvoiceNumber == "" {
			i.InvoiceNumber = fmt.Sprintf("INV-%06d", i.ID)

			if err := withInvoiceInternalSave(tx).Save(i).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (i *Invoice) MarkPaid(transactionID string) error {
	if i.ID == 0 {
		return ErrInvoiceNotFound
	}

	transactionID = strings.TrimSpace(transactionID)

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var invoice Invoice

		if err := tx.Clauses(
			clause.Locking{Strength: "UPDATE"},
		).First(&invoice, i.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceNotFound
			}

			return err
		}

		if invoice.Paid {
			return ErrInvoiceAlreadyPaid
		}

		now := time.Now()

		invoice.Paid = true
		invoice.PaidDate = &now
		invoice.TransactionID = transactionID

		if err := withInvoiceInternalSave(tx).Save(&invoice).Error; err != nil {
			return err
		}

		i.InvoiceNumber = invoice.InvoiceNumber
		i.TransactionID = invoice.TransactionID
		i.InvoiceDate = invoice.InvoiceDate
		i.DueDate = invoice.DueDate
		i.PaidDate = invoice.PaidDate
		i.StudentID = invoice.StudentID
		i.EnrollmentID = invoice.EnrollmentID
		i.Amount = invoice.Amount
		i.Paid = invoice.Paid

		return nil
	})

	if err != nil {
		switch {
		case errors.Is(err, ErrInvoiceNotFound),
			errors.Is(err, ErrInvoiceAlreadyPaid):
			return err
		default:
			return err
		}
	}

	return nil
}
