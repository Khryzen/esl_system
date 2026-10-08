package models

import (
	"errors"
	"testing"
	"time"

	"github.com/uadmin/uadmin"
)

func setupPaymentTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/payment_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&uadmin.User{},
		&Invoice{},
		&Payment{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})
}

func createPaymentTestInvoice(t *testing.T, amount float64) Invoice {
	t.Helper()

	invoice := Invoice{
		InvoiceNumber: "INV-PAY-001",
		InvoiceDate:   time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 14),
		StudentID:     10,
		EnrollmentID:  20,
		Amount:        amount,
		Paid:          false,
	}

	if err := invoice.Create(); err != nil {
		t.Fatalf("invoice.Create() error = %v", err)
	}

	return invoice
}

func TestPaymentCreate(t *testing.T) {
	setupPaymentTestDB(t)

	invoice := createPaymentTestInvoice(t, 10000)

	paymentDate := time.Date(
		2026,
		10,
		8,
		10,
		30,
		0,
		0,
		time.Local,
	)

	payment := Payment{
		InvoiceID:       invoice.ID,
		Amount:          5000,
		PaymentDate:     paymentDate,
		PaymentMethod:   PaymentMethodGCash,
		ReferenceNumber: "GCASH-12345",
		Notes:           "Initial payment",
	}

	if err := payment.Create(); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if payment.ID == 0 {
		t.Fatal("payment ID = 0, want persisted payment")
	}

	var saved Payment

	if err := uadmin.GetDB().
		First(&saved, payment.ID).Error; err != nil {
		t.Fatalf("failed to reload payment: %v", err)
	}

	if saved.InvoiceID != invoice.ID {
		t.Fatalf(
			"InvoiceID = %d, want %d",
			saved.InvoiceID,
			invoice.ID,
		)
	}

	if saved.Amount != 5000 {
		t.Fatalf(
			"Amount = %v, want 5000",
			saved.Amount,
		)
	}

	if saved.PaymentMethod != PaymentMethodGCash {
		t.Fatalf(
			"PaymentMethod = %q, want %q",
			saved.PaymentMethod,
			PaymentMethodGCash,
		)
	}

	if saved.ReferenceNumber != "GCASH-12345" {
		t.Fatalf(
			"ReferenceNumber = %q, want %q",
			saved.ReferenceNumber,
			"GCASH-12345",
		)
	}

	if saved.Notes != "Initial payment" {
		t.Fatalf(
			"Notes = %q, want %q",
			saved.Notes,
			"Initial payment",
		)
	}
}

func TestPaymentCreateDefaultsPaymentDate(t *testing.T) {
	setupPaymentTestDB(t)

	invoice := createPaymentTestInvoice(t, 10000)

	before := time.Now().In(time.Local)

	payment := Payment{
		InvoiceID:     invoice.ID,
		Amount:        1000,
		PaymentMethod: PaymentMethodCash,
	}

	if err := payment.Create(); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	after := time.Now().In(time.Local)

	if payment.PaymentDate.Before(before) ||
		payment.PaymentDate.After(after) {
		t.Fatalf(
			"PaymentDate = %v, want between %v and %v",
			payment.PaymentDate,
			before,
			after,
		)
	}
}

func TestPaymentCreateRejectsMissingInvoice(t *testing.T) {
	setupPaymentTestDB(t)

	payment := Payment{
		Amount:        1000,
		PaymentMethod: PaymentMethodCash,
	}

	err := payment.Create()

	if !errors.Is(err, ErrPaymentInvoiceRequired) {
		t.Fatalf(
			"Create() error = %v, want %v",
			err,
			ErrPaymentInvoiceRequired,
		)
	}
}

func TestPaymentCreateRejectsInvalidAmount(t *testing.T) {
	setupPaymentTestDB(t)

	invoice := createPaymentTestInvoice(t, 10000)

	tests := []struct {
		name   string
		amount float64
	}{
		{
			name:   "zero",
			amount: 0,
		},
		{
			name:   "negative",
			amount: -100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := Payment{
				InvoiceID:     invoice.ID,
				Amount:        tt.amount,
				PaymentMethod: PaymentMethodCash,
			}

			err := payment.Create()

			if !errors.Is(err, ErrPaymentAmountRequired) {
				t.Fatalf(
					"Create() error = %v, want %v",
					err,
					ErrPaymentAmountRequired,
				)
			}
		})
	}
}

func TestPaymentCreateRejectsInvalidPaymentMethod(t *testing.T) {
	setupPaymentTestDB(t)

	invoice := createPaymentTestInvoice(t, 10000)

	payment := Payment{
		InvoiceID:     invoice.ID,
		Amount:        1000,
		PaymentMethod: "PayPal",
	}

	err := payment.Create()

	if !errors.Is(err, ErrPaymentMethodInvalid) {
		t.Fatalf(
			"Create() error = %v, want %v",
			err,
			ErrPaymentMethodInvalid,
		)
	}
}

func TestPaymentCreateAcceptsAllPaymentMethods(t *testing.T) {
	setupPaymentTestDB(t)

	invoice := createPaymentTestInvoice(t, 10000)

	for _, method := range PaymentMethods() {
		t.Run(method, func(t *testing.T) {
			payment := Payment{
				InvoiceID:     invoice.ID,
				Amount:        100,
				PaymentMethod: method,
			}

			if err := payment.Create(); err != nil {
				t.Fatalf(
					"Create() error = %v for method %q",
					err,
					method,
				)
			}
		})
	}
}

func TestPaymentCreateRejectsNonexistentInvoice(t *testing.T) {
	setupPaymentTestDB(t)

	payment := Payment{
		InvoiceID:     99999,
		Amount:        1000,
		PaymentMethod: PaymentMethodCash,
	}

	err := payment.Create()

	if err == nil {
		t.Fatal("Create() error = nil, want error")
	}
}
