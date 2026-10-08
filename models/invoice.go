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
	ErrInvoiceNotFound = errors.New(
		"invoice not found",
	)

	ErrInvoiceAlreadyPaid = errors.New(
		"invoice is already paid",
	)

	ErrInvoicePaymentAmountRequired = errors.New(
		"The payment amount must be greater than zero.",
	)

	ErrInvoicePaymentExceedsBalance = errors.New(
		"The payment amount exceeds the remaining invoice balance.",
	)
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
	db := uadmin.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		return i.CreateWithTx(tx)
	})
}

func (i *Invoice) CreateWithTx(tx *gorm.DB) error {
	if i.ID != 0 {
		return errors.New("invoice already exists")
	}

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

func (i *Invoice) TotalPaid() (float64, error) {
	if i.ID == 0 {
		return 0, ErrInvoiceNotFound
	}

	var total float64

	if err := uadmin.GetDB().
		Model(&Payment{}).
		Where("invoice_id = ?", i.ID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (i *Invoice) Balance() (float64, error) {
	if i.ID == 0 {
		return 0, ErrInvoiceNotFound
	}

	totalPaid, err := i.TotalPaid()
	if err != nil {
		return 0, err
	}

	balance := i.Amount - totalPaid

	if balance < 0 {
		return 0, nil
	}

	return balance, nil
}

func (i *Invoice) PaymentStatus() (string, error) {
	if i.ID == 0 {
		return "", ErrInvoiceNotFound
	}

	balance, err := i.Balance()
	if err != nil {
		return "", err
	}

	if balance <= 0 {
		return "Paid", nil
	}

	totalPaid, err := i.TotalPaid()
	if err != nil {
		return "", err
	}

	if totalPaid > 0 {
		return "Partially Paid", nil
	}

	return "Unpaid", nil
}

func (i *Invoice) RecordPayment(
	amount float64,
	paymentDate time.Time,
	paymentMethod string,
	referenceNumber string,
	notes string,
) (*Payment, error) {
	if i.ID == 0 {
		return nil, ErrInvoiceNotFound
	}

	db := uadmin.GetDB()

	var payment *Payment

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error

		payment, err = i.RecordPaymentWithTx(
			tx,
			amount,
			paymentDate,
			paymentMethod,
			referenceNumber,
			notes,
		)

		return err
	})

	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (i *Invoice) RecordPaymentWithTx(
	tx *gorm.DB,
	amount float64,
	paymentDate time.Time,
	paymentMethod string,
	referenceNumber string,
	notes string,
) (*Payment, error) {
	if i.ID == 0 {
		return nil, ErrInvoiceNotFound
	}

	if amount <= 0 {
		return nil, ErrInvoicePaymentAmountRequired
	}

	var invoice Invoice

	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&invoice, i.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}

		return nil, err
	}

	var totalPaid float64

	if err := tx.
		Model(&Payment{}).
		Where("invoice_id = ?", invoice.ID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalPaid).Error; err != nil {
		return nil, err
	}

	balance := invoice.Amount - totalPaid

	if balance <= 0 {
		return nil, ErrInvoiceAlreadyPaid
	}

	if amount > balance {
		return nil, ErrInvoicePaymentExceedsBalance
	}

	payment := &Payment{
		InvoiceID:       invoice.ID,
		Amount:          amount,
		PaymentDate:     paymentDate,
		PaymentMethod:   strings.TrimSpace(paymentMethod),
		ReferenceNumber: strings.TrimSpace(referenceNumber),
		Notes:           strings.TrimSpace(notes),
	}

	if err := payment.CreateWithTx(tx); err != nil {
		return nil, err
	}

	newTotalPaid := totalPaid + amount

	if newTotalPaid >= invoice.Amount {
		paidDate := paymentDate

		invoice.Paid = true
		invoice.PaidDate = &paidDate

		if err := withInvoiceInternalSave(tx).Save(&invoice).Error; err != nil {
			return nil, err
		}
	}

	i.Paid = invoice.Paid
	i.PaidDate = invoice.PaidDate

	return payment, nil
}
