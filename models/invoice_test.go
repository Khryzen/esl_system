package models

import (
	"errors"
	"fmt"
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

	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})
}

func TestInvoiceCreateIntegrity(t *testing.T) {
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

	if err := invoice.Create(); err != nil {
		t.Fatalf("initial Create() error = %v", err)
	}

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

	if err := invoice.Create(); err != nil {
		t.Fatalf("initial Create() error = %v", err)
	}

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

	if err := invoice.Create(); err != nil {
		t.Fatalf("initial Create() error = %v", err)
	}
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

func TestInvoiceCreateGeneratesInvoiceNumber(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	invoice := Invoice{
		InvoiceDate: time.Date(
			2026,
			10,
			1,
			10,
			0,
			0,
			0,
			time.Local,
		),
		DueDate: time.Date(
			2026,
			10,
			15,
			10,
			0,
			0,
			0,
			time.Local,
		),
		StudentID:    10,
		EnrollmentID: 20,
		Amount:       1500,
		Paid:         false,
	}

	if err := invoice.Create(); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if invoice.ID == 0 {
		t.Fatal("invoice ID = 0, want persisted invoice")
	}

	if invoice.InvoiceNumber == "" {
		t.Fatal("InvoiceNumber is empty, want generated invoice number")
	}

	want := fmt.Sprintf("INV-%06d", invoice.ID)

	if invoice.InvoiceNumber != want {
		t.Fatalf(
			"InvoiceNumber = %q, want %q",
			invoice.InvoiceNumber,
			want,
		)
	}

	var saved Invoice

	if err := uadmin.Get(
		&saved,
		"id = ?",
		invoice.ID,
	); err != nil {
		t.Fatalf("failed to reload invoice: %v", err)
	}

	if saved.InvoiceNumber != want {
		t.Fatalf(
			"persisted InvoiceNumber = %q, want %q",
			saved.InvoiceNumber,
			want,
		)
	}
}

func TestInvoiceCreatePreservesExistingInvoiceNumber(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	invoice := Invoice{
		InvoiceNumber: "CUSTOM-001",
		InvoiceDate:   time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 14),
		StudentID:     10,
		EnrollmentID:  20,
		Amount:        1500,
		Paid:          false,
	}

	if err := invoice.Create(); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if invoice.ID == 0 {
		t.Fatal("invoice ID = 0, want persisted invoice")
	}

	if invoice.InvoiceNumber != "CUSTOM-001" {
		t.Fatalf(
			"InvoiceNumber = %q, want %q",
			invoice.InvoiceNumber,
			"CUSTOM-001",
		)
	}

	var saved Invoice

	if err := uadmin.Get(
		&saved,
		"id = ?",
		invoice.ID,
	); err != nil {
		t.Fatalf("failed to reload invoice: %v", err)
	}

	if saved.InvoiceNumber != "CUSTOM-001" {
		t.Fatalf(
			"persisted InvoiceNumber = %q, want %q",
			saved.InvoiceNumber,
			"CUSTOM-001",
		)
	}
}

func TestInvoiceCreateRejectsExistingInvoice(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	invoice := Invoice{
		InvoiceNumber: "INV-EXISTING",
		InvoiceDate:   time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 14),
		StudentID:     10,
		EnrollmentID:  20,
		Amount:        1500,
		Paid:          false,
	}

	if err := invoice.Create(); err != nil {
		t.Fatalf("initial Create() error = %v", err)
	}

	originalID := invoice.ID

	err := invoice.Create()

	if err == nil {
		t.Fatal("second Create() error = nil, want error")
	}

	if invoice.ID != originalID {
		t.Fatalf(
			"invoice ID = %d, want unchanged ID %d",
			invoice.ID,
			originalID,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&Invoice{}).
		Where("id = ?", originalID).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count invoice: %v", err)
	}

	if count != 1 {
		t.Fatalf(
			"invoice count = %d, want 1",
			count,
		)
	}
}

func TestInvoiceRecordPaymentPartialAndFull(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	if err := uadmin.GetDB().AutoMigrate(&Payment{}); err != nil {
		t.Fatalf("AutoMigrate(Payment) error = %v", err)
	}

	invoice := createPaymentTestInvoice(t, 10000)

	firstPayment, err := invoice.RecordPayment(
		5000,
		PaymentMethodGCash,
		"GCASH-001",
		"First payment",
	)
	if err != nil {
		t.Fatalf("first RecordPayment() error = %v", err)
	}

	if firstPayment == nil {
		t.Fatal("first payment = nil, want payment")
	}

	totalPaid, err := invoice.TotalPaid()
	if err != nil {
		t.Fatalf("TotalPaid() error = %v", err)
	}

	if totalPaid != 5000 {
		t.Fatalf("TotalPaid() = %v, want 5000", totalPaid)
	}

	balance, err := invoice.Balance()
	if err != nil {
		t.Fatalf("Balance() error = %v", err)
	}

	if balance != 5000 {
		t.Fatalf("Balance() = %v, want 5000", balance)
	}

	status, err := invoice.PaymentStatus()
	if err != nil {
		t.Fatalf("PaymentStatus() error = %v", err)
	}

	if status != "Partially Paid" {
		t.Fatalf(
			"PaymentStatus() = %q, want %q",
			status,
			"Partially Paid",
		)
	}

	if invoice.Paid {
		t.Fatal("invoice.Paid = true after partial payment, want false")
	}

	secondPayment, err := invoice.RecordPayment(
		5000,
		PaymentMethodCash,
		"",
		"Final payment",
	)
	if err != nil {
		t.Fatalf("second RecordPayment() error = %v", err)
	}

	if secondPayment == nil {
		t.Fatal("second payment = nil, want payment")
	}

	totalPaid, err = invoice.TotalPaid()
	if err != nil {
		t.Fatalf("TotalPaid() after final payment error = %v", err)
	}

	if totalPaid != 10000 {
		t.Fatalf("TotalPaid() = %v, want 10000", totalPaid)
	}

	balance, err = invoice.Balance()
	if err != nil {
		t.Fatalf("Balance() after final payment error = %v", err)
	}

	if balance != 0 {
		t.Fatalf("Balance() = %v, want 0", balance)
	}

	status, err = invoice.PaymentStatus()
	if err != nil {
		t.Fatalf("PaymentStatus() after final payment error = %v", err)
	}

	if status != "Paid" {
		t.Fatalf(
			"PaymentStatus() = %q, want %q",
			status,
			"Paid",
		)
	}

	if !invoice.Paid {
		t.Fatal("invoice.Paid = false after final payment, want true")
	}

	if invoice.PaidDate == nil {
		t.Fatal("invoice.PaidDate = nil, want final payment date")
	}

}

func TestInvoiceRecordPaymentRejectsOverpayment(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	if err := uadmin.GetDB().AutoMigrate(&Payment{}); err != nil {
		t.Fatalf("AutoMigrate(Payment) error = %v", err)
	}

	invoice := createPaymentTestInvoice(t, 10000)

	_, err := invoice.RecordPayment(
		10001,
		PaymentMethodCash,
		"",
		"",
	)

	if !errors.Is(err, ErrInvoicePaymentExceedsBalance) {
		t.Fatalf(
			"RecordPayment() error = %v, want %v",
			err,
			ErrInvoicePaymentExceedsBalance,
		)
	}

	totalPaid, err := invoice.TotalPaid()
	if err != nil {
		t.Fatalf("TotalPaid() error = %v", err)
	}

	if totalPaid != 0 {
		t.Fatalf(
			"TotalPaid() = %v after rejected payment, want 0",
			totalPaid,
		)
	}
}

func TestInvoiceRecordPaymentRejectsPaymentAfterPaid(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	if err := uadmin.GetDB().AutoMigrate(&Payment{}); err != nil {
		t.Fatalf("AutoMigrate(Payment) error = %v", err)
	}

	invoice := createPaymentTestInvoice(t, 10000)

	if _, err := invoice.RecordPayment(
		10000,
		PaymentMethodBankTransfer,
		"BANK-001",
		"",
	); err != nil {
		t.Fatalf("initial RecordPayment() error = %v", err)
	}

	_, err := invoice.RecordPayment(
		1,
		PaymentMethodCash,
		"",
		"",
	)

	if !errors.Is(err, ErrInvoiceAlreadyPaid) {
		t.Fatalf(
			"RecordPayment() error = %v, want %v",
			err,
			ErrInvoiceAlreadyPaid,
		)
	}

	totalPaid, err := invoice.TotalPaid()
	if err != nil {
		t.Fatalf("TotalPaid() error = %v", err)
	}

	if totalPaid != 10000 {
		t.Fatalf(
			"TotalPaid() = %v, want 10000",
			totalPaid,
		)
	}
}

func TestInvoiceRecordPaymentRejectsInvalidAmount(t *testing.T) {
	setupInvoiceSaveIntegrityTestDB(t)

	if err := uadmin.GetDB().AutoMigrate(&Payment{}); err != nil {
		t.Fatalf("AutoMigrate(Payment) error = %v", err)
	}

	invoice := createPaymentTestInvoice(t, 10000)

	for _, amount := range []float64{0, -1} {
		_, err := invoice.RecordPayment(
			amount,
			PaymentMethodCash,
			"",
			"",
		)

		if !errors.Is(err, ErrInvoicePaymentAmountRequired) {
			t.Fatalf(
				"RecordPayment(%v) error = %v, want %v",
				amount,
				err,
				ErrInvoicePaymentAmountRequired,
			)
		}
	}
}
