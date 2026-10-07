package views

import (
	"encoding/json"
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
		&models.Student{},
		&models.Course{},
		&models.Package{},
		&models.Enrollment{},
		&models.Invoice{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})
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

	if err := invoice.Create(); err != nil {
		t.Fatalf("invoice Create() error = %v", err)
	}

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

func TestCreateInvoice(t *testing.T) {
	setupInvoiceHandlerTestDB(t)

	enrollment := models.Enrollment{
		StudentID:        10,
		CourseID:         20,
		PackageID:        30,
		TotalClasses:     10,
		ClassesRemaining: 10,
		ReferenceNumber:  "ENR-001",
		Active:           true,
	}

	if err := uadmin.GetDB().Create(&enrollment).Error; err != nil {
		t.Fatalf("failed to create enrollment: %v", err)
	}

	form := url.Values{}
	form.Set("enrollment_id", strconv.FormatUint(uint64(enrollment.ID), 10))
	form.Set("amount", "1500.50")
	form.Set("due_date", "2026-10-15")

	req := httptest.NewRequest(
		http.MethodPost,
		"/invoice",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	createInvoice(rec, req)

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
		"enrollment_id = ?",
		enrollment.ID,
	); err != nil {
		t.Fatalf("failed to load created invoice: %v", err)
	}

	if saved.StudentID != enrollment.StudentID {
		t.Fatalf(
			"StudentID = %d, want %d",
			saved.StudentID,
			enrollment.StudentID,
		)
	}

	if saved.EnrollmentID != enrollment.ID {
		t.Fatalf(
			"EnrollmentID = %d, want %d",
			saved.EnrollmentID,
			enrollment.ID,
		)
	}

	if saved.Amount != 1500.50 {
		t.Fatalf(
			"Amount = %v, want 1500.50",
			saved.Amount,
		)
	}

	if saved.Paid {
		t.Fatal("Paid = true, want false")
	}

	if saved.InvoiceNumber == "" {
		t.Fatal("InvoiceNumber is empty, want generated invoice number")
	}
}

func TestCreateInvoiceRejectsInvalidAmount(t *testing.T) {
	setupInvoiceHandlerTestDB(t)

	enrollment := models.Enrollment{
		StudentID:        10,
		CourseID:         20,
		PackageID:        30,
		TotalClasses:     10,
		ClassesRemaining: 10,
		ReferenceNumber:  "ENR-INVALID-AMOUNT",
		Active:           true,
	}

	if err := uadmin.GetDB().Create(&enrollment).Error; err != nil {
		t.Fatalf("failed to create enrollment: %v", err)
	}

	form := url.Values{}
	form.Set(
		"enrollment_id",
		strconv.FormatUint(uint64(enrollment.ID), 10),
	)
	form.Set("amount", "0")
	form.Set("due_date", "2026-10-15")

	req := httptest.NewRequest(
		http.MethodPost,
		"/invoice",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	createInvoice(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d; body = %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"Enter an amount greater than 0.",
	) {
		t.Fatalf(
			"response body = %q, want invalid amount message",
			rec.Body.String(),
		)
	}
}

func TestMarkInvoicePaidResponse(t *testing.T) {
	setupInvoiceHandlerTestDB(t)

	invoice := createInvoiceHandlerTestInvoice(t, false)

	form := url.Values{}
	form.Set("invoice_id", strconv.FormatUint(uint64(invoice.ID), 10))
	form.Set("transaction_id", "TXN-20261007-001")

	req := httptest.NewRequest(
		http.MethodPut,
		"/invoice",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	markInvoicePaid(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("markInvoicePaid() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response struct {
		Status        string     `json:"status"`
		InvoiceID     uint       `json:"invoice_id"`
		TransactionID string     `json:"transaction_id"`
		Paid          bool       `json:"paid"`
		PaidDate      *time.Time `json:"paid_date"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Fatalf("response status = %q, want %q", response.Status, "ok")
	}

	if response.InvoiceID != invoice.ID {
		t.Fatalf(
			"response invoice_id = %d, want %d",
			response.InvoiceID,
			invoice.ID,
		)
	}

	if response.TransactionID != "TXN-20261007-001" {
		t.Fatalf(
			"response transaction_id = %q, want %q",
			response.TransactionID,
			"TXN-20261007-001",
		)
	}

	if !response.Paid {
		t.Fatal("response paid = false, want true")
	}

	if response.PaidDate == nil {
		t.Fatal("response paid_date = nil, want a payment date")
	}
}

func TestInvoiceDetailsHandler(t *testing.T) {
	setupInvoiceHandlerTestDB(t)

	validFrom := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	validUntil := time.Date(2026, 12, 31, 23, 59, 59, 0, time.Local)

	student := models.Student{
		FirstName: "John",
		LastName:  "Doe",
		WeChatID:  "john-doe",
	}

	if err := uadmin.GetDB().Create(&student).Error; err != nil {
		t.Fatalf("failed to create student: %v", err)
	}

	course := models.Course{
		Title:  "General English",
		Active: true,
	}

	if err := uadmin.GetDB().Create(&course).Error; err != nil {
		t.Fatalf("failed to create course: %v", err)
	}

	pkg := models.Package{
		Name:                   "10 Class Package",
		NumberOfClasses:        10,
		NumberOfFreeClasses:    0,
		TotalClasses:           10,
		ClassDurationInMinutes: 50,
		Price:                  1500,
		ValidFrom:              &validFrom,
		ValidUntil:             &validUntil,
		Active:                 true,
	}

	if err := uadmin.GetDB().Create(&pkg).Error; err != nil {
		t.Fatalf("failed to create package: %v", err)
	}

	enrollment := models.Enrollment{
		StudentID:        student.ID,
		CourseID:         course.ID,
		PackageID:        pkg.ID,
		TotalClasses:     10,
		ClassesRemaining: 7,
		ReferenceNumber:  "ENR-DETAILS-001",
		Active:           true,
	}

	if err := uadmin.GetDB().Create(&enrollment).Error; err != nil {
		t.Fatalf("failed to create enrollment: %v", err)
	}

	invoiceDate := time.Date(2026, 10, 5, 10, 0, 0, 0, time.Local)
	dueDate := time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local)
	paidDate := time.Date(2026, 10, 6, 15, 0, 0, 0, time.Local)

	invoice := models.Invoice{
		InvoiceNumber: "INV-DETAILS-001",
		TransactionID: "TX-DETAILS-001",
		InvoiceDate:   invoiceDate,
		DueDate:       dueDate,
		PaidDate:      &paidDate,
		StudentID:     student.ID,
		EnrollmentID:  enrollment.ID,
		Amount:        1500,
		Paid:          true,
	}

	if err := uadmin.GetDB().Create(&invoice).Error; err != nil {
		t.Fatalf("failed to create invoice: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/invoice/details?id="+strconv.FormatUint(uint64(invoice.ID), 10),
		nil,
	)

	rec := httptest.NewRecorder()

	InvoiceDetailsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d; body = %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	var response invoiceDetailsResponse

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v; body = %s", err, rec.Body.String())
	}

	if response.ID != invoice.ID {
		t.Fatalf("ID = %d, want %d", response.ID, invoice.ID)
	}

	if response.InvoiceNumber != "INV-DETAILS-001" {
		t.Fatalf(
			"InvoiceNumber = %q, want %q",
			response.InvoiceNumber,
			"INV-DETAILS-001",
		)
	}

	if !response.InvoiceDate.Equal(invoiceDate) {
		t.Fatalf(
			"InvoiceDate = %v, want %v",
			response.InvoiceDate,
			invoiceDate,
		)
	}

	if !response.DueDate.Equal(dueDate) {
		t.Fatalf(
			"DueDate = %v, want %v",
			response.DueDate,
			dueDate,
		)
	}

	if response.Amount != 1500 {
		t.Fatalf("Amount = %v, want 1500", response.Amount)
	}

	if !response.Paid {
		t.Fatal("Paid = false, want true")
	}

	if response.TransactionID != "TX-DETAILS-001" {
		t.Fatalf(
			"TransactionID = %q, want %q",
			response.TransactionID,
			"TX-DETAILS-001",
		)
	}

	if response.PaidDate == nil {
		t.Fatal("PaidDate = nil, want populated payment date")
	}

	if !response.PaidDate.Equal(paidDate) {
		t.Fatalf(
			"PaidDate = %v, want %v",
			*response.PaidDate,
			paidDate,
		)
	}

	if response.StudentID != student.ID {
		t.Fatalf(
			"StudentID = %d, want %d",
			response.StudentID,
			student.ID,
		)
	}

	if response.StudentName != "John Doe" {
		t.Fatalf(
			"StudentName = %q, want %q",
			response.StudentName,
			"John Doe",
		)
	}

	if response.EnrollmentID != enrollment.ID {
		t.Fatalf(
			"EnrollmentID = %d, want %d",
			response.EnrollmentID,
			enrollment.ID,
		)
	}

	if response.EnrollmentRef != "ENR-DETAILS-001" {
		t.Fatalf(
			"EnrollmentRef = %q, want %q",
			response.EnrollmentRef,
			"ENR-DETAILS-001",
		)
	}

	if response.CourseID != course.ID {
		t.Fatalf(
			"CourseID = %d, want %d",
			response.CourseID,
			course.ID,
		)
	}

	if response.CourseName != "General English" {
		t.Fatalf(
			"CourseName = %q, want %q",
			response.CourseName,
			"General English",
		)
	}

	if response.PackageID != pkg.ID {
		t.Fatalf(
			"PackageID = %d, want %d",
			response.PackageID,
			pkg.ID,
		)
	}

	if response.PackageName != "10 Class Package" {
		t.Fatalf(
			"PackageName = %q, want %q",
			response.PackageName,
			"10 Class Package",
		)
	}
}

func TestInvoiceDetailsHandlerRejectsInvalidID(t *testing.T) {
	setupInvoiceHandlerTestDB(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/invoice/details?id=invalid",
		nil,
	)

	rec := httptest.NewRecorder()

	InvoiceDetailsHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"status code = %d, want %d; body = %s",
			rec.Code,
			http.StatusBadRequest,
			rec.Body.String(),
		)
	}

	if !strings.Contains(rec.Body.String(), "Invalid invoice ID") {
		t.Fatalf(
			"response body = %q, want invalid invoice ID message",
			rec.Body.String(),
		)
	}
}

func TestInvoiceDetailsHandlerNotFound(t *testing.T) {
	setupInvoiceHandlerTestDB(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/invoice/details?id=999999",
		nil,
	)

	rec := httptest.NewRecorder()

	InvoiceDetailsHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"status code = %d, want %d; body = %s",
			rec.Code,
			http.StatusNotFound,
			rec.Body.String(),
		)
	}

	if !strings.Contains(rec.Body.String(), "Invoice not found") {
		t.Fatalf(
			"response body = %q, want invoice-not-found message",
			rec.Body.String(),
		)
	}
}
