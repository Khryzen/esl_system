package views

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func setupInvoiceHandlerTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/invoice_handler_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&uadmin.User{},
		&models.Invoice{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
}

func createInvoiceHandlerTestInvoice(t *testing.T, paid bool) models.Invoice {
	t.Helper()

	invoiceDate := time.Date(
		2026,
		10,
		1,
		10,
		0,
		0,
		0,
		time.Local,
	)

	dueDate := time.Date(
		2026,
		10,
		15,
		10,
		0,
		0,
		0,
		time.Local,
	)

	invoice := models.Invoice{
		InvoiceNumber: "INV-000001",
		InvoiceDate:   invoiceDate,
		DueDate:       dueDate,
		StudentID:     10,
		EnrollmentID:  20,
		Amount:        1500,
		Paid:          paid,
	}

	if paid {
		paidDate := time.Date(
			2026,
			10,
			2,
			15,
			0,
			0,
			0,
			time.Local,
		)

		invoice.PaidDate = &paidDate
		invoice.TransactionID = "TX-ORIGINAL"
	}

	invoice.Save()

	if invoice.ID == 0 {
		t.Fatal("invoice ID = 0, want persisted invoice")
	}

	return invoice
}

func TestMarkInvoicePaid(t *testing.T) {
	setupInvoiceHandlerTestDB(t)

	invoice := createInvoiceHandlerTestInvoice(t, false)

	form := url.Values{}
	form.Set("invoice_id", strconv.FormatUint(uint64(invoice.ID), 10))
	form.Set("transaction_id", "TX-12345")

	req := httptest.NewRequest(
		http.MethodPut,
		"/invoice",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	markInvoicePaid(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d; body = %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	var saved models.Invoice

	if err := uadmin.Get(
		&saved,
		"id = ?",
		invoice.ID,
	); err != nil {
		t.Fatalf("failed to reload invoice: %v", err)
	}

	if !saved.Paid {
		t.Fatal("Paid = false, want true")
	}

	if saved.TransactionID != "TX-12345" {
		t.Fatalf(
			"TransactionID = %q, want %q",
			saved.TransactionID,
			"TX-12345",
		)
	}

	if saved.PaidDate == nil {
		t.Fatal("PaidDate = nil, want populated payment date")
	}
}

func TestMarkInvoicePaidRejectsAlreadyPaid(t *testing.T) {
	setupInvoiceHandlerTestDB(t)

	invoice := createInvoiceHandlerTestInvoice(t, true)

	form := url.Values{}
	form.Set("invoice_id", strconv.FormatUint(uint64(invoice.ID), 10))
	form.Set("transaction_id", "TX-NEW")

	req := httptest.NewRequest(
		http.MethodPut,
		"/invoice",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	markInvoicePaid(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d; body = %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	if !strings.Contains(rec.Body.String(), "already paid") {
		t.Fatalf(
			"response body = %q, want already-paid error",
			rec.Body.String(),
		)
	}

	var saved models.Invoice

	if err := uadmin.Get(
		&saved,
		"id = ?",
		invoice.ID,
	); err != nil {
		t.Fatalf("failed to reload invoice: %v", err)
	}

	if saved.TransactionID != "TX-ORIGINAL" {
		t.Fatalf(
			"TransactionID = %q, want original transaction ID %q",
			saved.TransactionID,
			"TX-ORIGINAL",
		)
	}
}

func TestMarkInvoicePaidRejectsMissingInvoice(t *testing.T) {
	setupInvoiceHandlerTestDB(t)

	form := url.Values{}
	form.Set("invoice_id", "999999")
	form.Set("transaction_id", "TX-12345")

	req := httptest.NewRequest(
		http.MethodPut,
		"/invoice",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	markInvoicePaid(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d; body = %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	if !strings.Contains(rec.Body.String(), "Invoice not found") {
		t.Fatalf(
			"response body = %q, want invoice-not-found error",
			rec.Body.String(),
		)
	}
}
