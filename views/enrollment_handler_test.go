package views

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

func setupEnrollmentHandlerTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/enrollment_handler_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&uadmin.User{},
		&models.Student{},
		&models.Course{},
		&models.Level{},
		&models.Package{},
		&models.Enrollment{},
		&models.Invoice{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
}

func createEnrollmentHandlerTestStudent(t *testing.T) models.Student {
	t.Helper()

	student := models.Student{
		FirstName: "Test",
		LastName:  "Student",
		Email:     "test@example.com",
		WeChatID:  "test-wechat",
	}

	if _, err := student.Create(); err != nil {
		t.Fatalf("student Create() error = %v", err)
	}

	if student.ID == 0 {
		t.Fatal("student ID = 0, want persisted student")
	}

	return student
}

func createEnrollmentHandlerTestCourse(t *testing.T) models.Course {
	t.Helper()

	level := models.Level{
		Level: "Beginner",
	}

	if err := uadmin.Save(&level); err != nil {
		t.Fatalf("failed to create level: %v", err)
	}

	course := models.Course{
		Title:   "General English",
		LevelID: level.ID,
		Active:  true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("failed to create course: %v", err)
	}

	if course.ID == 0 {
		t.Fatal("course ID = 0, want persisted course")
	}

	return course
}

func createEnrollmentHandlerTestPackage(t *testing.T) models.Package {
	t.Helper()

	now := time.Now()
	validFrom := now.AddDate(0, 0, -1)
	validUntil := now.AddDate(0, 0, 30)

	pkg := models.Package{
		Name:                "Test Package",
		NumberOfClasses:     10,
		NumberOfFreeClasses: 2,
		Price:               1500,
		ValidFrom:           &validFrom,
		ValidUntil:          &validUntil,
		Active:              true,
	}

	if err := pkg.Save(); err != nil {
		t.Fatalf("package Save() error = %v", err)
	}

	if pkg.ID == 0 {
		t.Fatal("package ID = 0, want persisted package")
	}

	return pkg
}

func enrollmentHandlerFormRequest(
	method string,
	target string,
	values url.Values,
) *http.Request {
	req := httptest.NewRequest(
		method,
		target,
		strings.NewReader(values.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	return req
}

func TestEnrollmentHandlerInvoiceFailureReturnsError(t *testing.T) {
	setupEnrollmentHandlerTestDB(t)
	db := uadmin.GetDB()

	student := models.Student{
		FirstName: "Invoice",
		LastName:  "Failure",
		Email:     "invoice.failure@example.com",
		WeChatID:  "invoice_failure",
	}
	if _, err := student.Create(); err != nil {
		t.Fatalf("create student: %v", err)
	}

	course := models.Course{
		Title:  "Invoice Failure Course",
		Active: true,
	}
	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("create course: %v", err)
	}

	validFrom := time.Now().Add(-24 * time.Hour)
	validUntil := time.Now().Add(24 * time.Hour)
	pkg := models.Package{
		Name:                "Invoice Failure Package",
		NumberOfClasses:     10,
		NumberOfFreeClasses: 0,
		Price:               100,
		Active:              true,
		ValidFrom:           &validFrom,
		ValidUntil:          &validUntil,
	}
	if err := pkg.Save(); err != nil {
		t.Fatalf("create package: %v", err)
	}

	originalCreateInvoice := createEnrollmentInvoice
	t.Cleanup(func() {
		createEnrollmentInvoice = originalCreateInvoice
	})

	createEnrollmentInvoice = func(invoice *models.Invoice) error {
		return errors.New("forced invoice failure")
	}

	form := url.Values{}
	form.Set("student_type", "existing")
	form.Set("StudentID", strconv.FormatUint(uint64(student.ID), 10))
	form.Set("CourseID", strconv.FormatUint(uint64(course.ID), 10))
	form.Set("PackageID", strconv.FormatUint(uint64(pkg.ID), 10))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, values := range form {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				t.Fatalf("write form field %s: %v", key, err)
			}
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/enrollment", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()

	createEnrollment(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v\nbody=%s", err, rec.Body.String())
	}

	if response["status"] != "error" {
		t.Fatalf("response status = %v, want error", response["status"])
	}

	message, ok := response["message"].(string)
	if !ok {
		t.Fatalf("response message = %v, want string", response["message"])
	}

	wantMessage := "The enrollment was created, but its invoice could not be created. Please retry creating the invoice from the Invoices page."
	if message != wantMessage {
		t.Fatalf("response message = %q, want %q", message, wantMessage)
	}

	var enrollment models.Enrollment
	if err := db.First(&enrollment).Error; err != nil {
		t.Fatalf("expected enrollment to exist: %v", err)
	}

	var invoice models.Invoice
	if err := db.First(&invoice).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("invoice should not exist, err = %v", err)
	}
}
