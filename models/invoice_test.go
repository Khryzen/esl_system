package models

import (
	"errors"
	"testing"
	"time"

	"github.com/uadmin/uadmin"
)

func setupInvoiceSaveIntegrityTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/invoice_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&uadmin.User{},
		&Invoice{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
}

func TestInvoiceSaveIntegrity(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	invoiceDate := time.Date(
		2026, 10, 1,
		10, 0, 0, 0,
		time.Local,
	)

	dueDate := time.Date(
		2026, 10, 15,
		10, 0, 0, 0,
		time.Local,
	)

	paidDate := time.Date(
		2026, 10, 2,
		15, 0, 0, 0,
		time.Local,
	)

	invoice := Invoice{
		InvoiceNumber: "INV-000001",
		TransactionID: "TX-001",
		InvoiceDate:   invoiceDate,
		DueDate:       dueDate,
		PaidDate:      &paidDate,
		StudentID:     10,
		EnrollmentID:  20,
		Amount:        1500,
		Paid:          true,
	}

	invoice.Save()

	if invoice.ID == 0 {
		t.Fatal("invoice ID = 0, want persisted invoice")
	}

	var existing Invoice

	if err := uadmin.Get(
		&existing,
		"id = ?",
		invoice.ID,
	); err != nil {
		t.Fatalf("failed to load invoice: %v", err)
	}

	originalInvoiceNumber := existing.InvoiceNumber
	originalTransactionID := existing.TransactionID
	originalInvoiceDate := existing.InvoiceDate
	originalDueDate := existing.DueDate
	originalStudentID := existing.StudentID
	originalEnrollmentID := existing.EnrollmentID
	originalAmount := existing.Amount
	originalPaid := existing.Paid

	var originalPaidDate time.Time

	if existing.PaidDate != nil {
		originalPaidDate = *existing.PaidDate
	}

	existing.InvoiceNumber = "HACKED-INVOICE"
	existing.TransactionID = "HACKED-TX"
	existing.InvoiceDate = invoiceDate.AddDate(1, 0, 0)
	existing.DueDate = dueDate.AddDate(1, 0, 0)
	existing.StudentID = 999
	existing.EnrollmentID = 999
	existing.Amount = 999999
	existing.Paid = false

	hackedPaidDate := paidDate.AddDate(1, 0, 0)
	existing.PaidDate = &hackedPaidDate

	if err := uadmin.Save(&existing); err != nil {
		t.Fatalf("generic Save() error = %v", err)
	}

	var saved Invoice

	if err := uadmin.Get(
		&saved,
		"id = ?",
		invoice.ID,
	); err != nil {
		t.Fatalf("failed to reload invoice: %v", err)
	}

	if saved.InvoiceNumber != originalInvoiceNumber {
		t.Fatalf(
			"InvoiceNumber = %q, want protected value %q",
			saved.InvoiceNumber,
			originalInvoiceNumber,
		)
	}

	if saved.TransactionID != originalTransactionID {
		t.Fatalf(
			"TransactionID = %q, want protected value %q",
			saved.TransactionID,
			originalTransactionID,
		)
	}

	if !saved.InvoiceDate.Equal(originalInvoiceDate) {
		t.Fatalf(
			"InvoiceDate = %v, want protected value %v",
			saved.InvoiceDate,
			originalInvoiceDate,
		)
	}

	if !saved.DueDate.Equal(originalDueDate) {
		t.Fatalf(
			"DueDate = %v, want protected value %v",
			saved.DueDate,
			originalDueDate,
		)
	}

	if saved.StudentID != originalStudentID {
		t.Fatalf(
			"StudentID = %d, want protected value %d",
			saved.StudentID,
			originalStudentID,
		)
	}

	if saved.EnrollmentID != originalEnrollmentID {
		t.Fatalf(
			"EnrollmentID = %d, want protected value %d",
			saved.EnrollmentID,
			originalEnrollmentID,
		)
	}

	if saved.Amount != originalAmount {
		t.Fatalf(
			"Amount = %v, want protected value %v",
			saved.Amount,
			originalAmount,
		)
	}

	if saved.Paid != originalPaid {
		t.Fatalf(
			"Paid = %v, want protected value %v",
			saved.Paid,
			originalPaid,
		)
	}

	if saved.PaidDate == nil {
		t.Fatal("PaidDate = nil, want protected paid date")
	}

	if !saved.PaidDate.Equal(originalPaidDate) {
		t.Fatalf(
			"PaidDate = %v, want protected value %v",
			*saved.PaidDate,
			originalPaidDate,
		)
	}
}

func TestInvoiceMarkPaid(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	invoice := Invoice{
		InvoiceNumber: "INV-000001",
		InvoiceDate:   time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 14),
		StudentID:     10,
		EnrollmentID:  20,
		Amount:        1500,
		Paid:          false,
	}

	invoice.Save()

	if invoice.ID == 0 {
		t.Fatal("invoice ID = 0, want persisted invoice")
	}

	if err := invoice.MarkPaid("TX-123"); err != nil {
		t.Fatalf("MarkPaid() error = %v", err)
	}

	var saved Invoice

	if err := uadmin.Get(&saved, "id = ?", invoice.ID); err != nil {
		t.Fatalf("failed to reload invoice: %v", err)
	}

	if !saved.Paid {
		t.Fatal("Paid = false, want true")
	}

	if saved.PaidDate == nil {
		t.Fatal("PaidDate = nil, want payment date")
	}

	if saved.TransactionID != "TX-123" {
		t.Fatalf(
			"TransactionID = %q, want %q",
			saved.TransactionID,
			"TX-123",
		)
	}
}

func TestInvoiceMarkPaidRejectsAlreadyPaid(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	paidDate := time.Now()

	invoice := Invoice{
		InvoiceNumber: "INV-000001",
		InvoiceDate:   time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 14),
		PaidDate:      &paidDate,
		StudentID:     10,
		EnrollmentID:  20,
		Amount:        1500,
		Paid:          true,
	}

	invoice.Save()

	err := invoice.MarkPaid("TX-NEW")

	if !errors.Is(err, ErrInvoiceAlreadyPaid) {
		t.Fatalf(
			"MarkPaid() error = %v, want ErrInvoiceAlreadyPaid",
			err,
		)
	}
}

func TestInvoiceMarkPaidNotFound(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	invoice := Invoice{
		Model: uadmin.Model{
			ID: 999,
		},
	}

	err := invoice.MarkPaid("TX-123")

	if !errors.Is(err, ErrInvoiceNotFound) {
		t.Fatalf(
			"MarkPaid() error = %v, want ErrInvoiceNotFound",
			err,
		)
	}
}
