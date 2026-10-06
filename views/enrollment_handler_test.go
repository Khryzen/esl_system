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

	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})
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

	originalCreateInvoice := createEnrollmentInvoiceWithTx
	t.Cleanup(func() {
		createEnrollmentInvoiceWithTx = originalCreateInvoice
	})

	createEnrollmentInvoiceWithTx = func(tx *gorm.DB, invoice *models.Invoice) error {
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

	wantMessage := "The enrollment could not be saved."
	if message != wantMessage {
		t.Fatalf("response message = %q, want %q", message, wantMessage)
	}

	var enrollment models.Enrollment
	if err := db.First(&enrollment).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("enrollment should have been rolled back, err = %v", err)
	}
	var invoice models.Invoice
	if err := db.First(&invoice).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("invoice should not exist, err = %v", err)
	}
}
func TestEnrollmentHandlerNewStudentRollsBackWhenEnrollmentFails(t *testing.T) {
	setupEnrollmentHandlerTestDB(t)
	db := uadmin.GetDB()

	course := createEnrollmentHandlerTestCourse(t)
	course.Active = false

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("deactivate course: %v", err)
	}

	pkg := createEnrollmentHandlerTestPackage(t)

	form := url.Values{}
	form.Set("student_type", "new")
	form.Set("NewStudentFirstName", "Rollback")
	form.Set("NewStudentLastName", "Student")
	form.Set("NewStudentWeChat", "rollback_wechat")
	form.Set("NewStudentEmail", "rollback@example.com")
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

	wantMessage := "The student, enrollment, and invoice could not be created. Please try again."
	if message != wantMessage {
		t.Fatalf("response message = %q, want %q", message, wantMessage)
	}

	var student models.Student
	if err := db.
		Where("we_chat_id = ?", "rollback_wechat").
		First(&student).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("student should have been rolled back, err = %v", err)
	}

	var user uadmin.User
	if err := db.
		Where("first_name = ? AND last_name = ?", "Rollback", "Student").
		First(&user).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("student user should have been rolled back, err = %v", err)
	}

	var enrollment models.Enrollment
	if err := db.First(&enrollment).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("enrollment should not exist, err = %v", err)
	}

	var invoice models.Invoice
	if err := db.First(&invoice).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("invoice should not exist, err = %v", err)
	}
}

func TestEnrollmentTransactionError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		isNewStudent bool
		want         string
	}{
		{
			name: "duplicate enrollment",
			err:  models.ErrEnrollmentAlreadyExists,
			want: "The student already has an active enrollment for this course.",
		},
		{
			name: "inactive course",
			err:  models.ErrEnrollmentCourseInactive,
			want: "The selected course is not available for enrollment.",
		},
		{
			name: "inactive package",
			err:  models.ErrEnrollmentPackageInactive,
			want: "The selected package is not available for enrollment.",
		},
		{
			name: "expired package",
			err:  models.ErrEnrollmentPackageExpired,
			want: "The selected package is outside its validity period.",
		},
		{
			name: "package has no classes",
			err:  models.ErrEnrollmentPackageNoClasses,
			want: "The selected package has no available classes.",
		},
		{
			name: "unexpected existing student error",
			err:  errors.New("database failure"),
			want: "The enrollment could not be saved.",
		},
		{
			name:         "unexpected new student error",
			err:          errors.New("database failure"),
			isNewStudent: true,
			want:         "The student, enrollment, and invoice could not be created. Please try again.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := enrollmentTransactionError(tt.err, tt.isNewStudent)

			if err == nil {
				t.Fatal("expected error")
			}

			if err.Error() != tt.want {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestEnrollmentChangeCourseHandler(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) (models.Enrollment, models.Course)
		requestCourse func(enrollment models.Enrollment, course models.Course) string
		wantStatus    string
		wantMessage   string
		wantCourseID  uint
	}{
		{
			name: "changes course successfully",
			setup: func(t *testing.T) (models.Enrollment, models.Course) {
				student := createEnrollmentHandlerTestStudent(t)
				firstCourse := createEnrollmentHandlerTestCourse(t)

				secondCourse := models.Course{
					Title:  "Business English",
					Active: true,
				}

				if err := uadmin.Save(&secondCourse); err != nil {
					t.Fatalf("create second course: %v", err)
				}

				pkg := createEnrollmentHandlerTestPackage(t)

				enrollment := models.Enrollment{
					StudentID: student.ID,
					CourseID:  firstCourse.ID,
					PackageID: pkg.ID,
				}

				if err := enrollment.Create(); err != nil {
					t.Fatalf("create enrollment: %v", err)
				}

				return enrollment, secondCourse
			},
			requestCourse: func(enrollment models.Enrollment, course models.Course) string {
				return strconv.FormatUint(uint64(course.ID), 10)
			},
			wantStatus:   "ok",
			wantCourseID: 0,
		},
		{
			name: "rejects invalid enrollment ID",
			setup: func(t *testing.T) (models.Enrollment, models.Course) {
				return models.Enrollment{}, models.Course{}
			},
			requestCourse: func(enrollment models.Enrollment, course models.Course) string {
				return strconv.FormatUint(uint64(course.ID), 10)
			},
			wantStatus:  "error",
			wantMessage: "Invalid enrollment ID.",
		},
		{
			name: "rejects invalid course ID",
			setup: func(t *testing.T) (models.Enrollment, models.Course) {
				return models.Enrollment{
					Model: uadmin.Model{
						ID: 1,
					},
				}, models.Course{}
			},
			requestCourse: func(enrollment models.Enrollment, course models.Course) string {
				return "invalid"
			},
			wantStatus:  "error",
			wantMessage: "Select a course.",
		},
		{
			name: "rejects missing enrollment",
			setup: func(t *testing.T) (models.Enrollment, models.Course) {
				course := createEnrollmentHandlerTestCourse(t)

				return models.Enrollment{
					Model: uadmin.Model{
						ID: 999999,
					},
				}, course
			},
			requestCourse: func(enrollment models.Enrollment, course models.Course) string {
				return strconv.FormatUint(uint64(course.ID), 10)
			},
			wantStatus:  "error",
			wantMessage: "Enrollment not found.",
		},
		{
			name: "rejects inactive course",
			setup: func(t *testing.T) (models.Enrollment, models.Course) {
				student := createEnrollmentHandlerTestStudent(t)
				firstCourse := createEnrollmentHandlerTestCourse(t)

				inactiveCourse := models.Course{
					Title:  "Inactive Course",
					Active: false,
				}

				if err := uadmin.Save(&inactiveCourse); err != nil {
					t.Fatalf("create inactive course: %v", err)
				}

				pkg := createEnrollmentHandlerTestPackage(t)

				enrollment := models.Enrollment{
					StudentID: student.ID,
					CourseID:  firstCourse.ID,
					PackageID: pkg.ID,
				}

				if err := enrollment.Create(); err != nil {
					t.Fatalf("create enrollment: %v", err)
				}

				return enrollment, inactiveCourse
			},
			requestCourse: func(enrollment models.Enrollment, course models.Course) string {
				return strconv.FormatUint(uint64(course.ID), 10)
			},
			wantStatus:  "error",
			wantMessage: "The selected course is not available.",
		},
		{
			name: "rejects same course",
			setup: func(t *testing.T) (models.Enrollment, models.Course) {
				student := createEnrollmentHandlerTestStudent(t)
				course := createEnrollmentHandlerTestCourse(t)
				pkg := createEnrollmentHandlerTestPackage(t)

				enrollment := models.Enrollment{
					StudentID: student.ID,
					CourseID:  course.ID,
					PackageID: pkg.ID,
				}

				if err := enrollment.Create(); err != nil {
					t.Fatalf("create enrollment: %v", err)
				}

				return enrollment, course
			},
			requestCourse: func(enrollment models.Enrollment, course models.Course) string {
				return strconv.FormatUint(uint64(course.ID), 10)
			},
			wantStatus:  "error",
			wantMessage: "The enrollment is already assigned to this course.",
		},
		{
			name: "rejects duplicate active enrollment",
			setup: func(t *testing.T) (models.Enrollment, models.Course) {
				student := createEnrollmentHandlerTestStudent(t)
				firstCourse := createEnrollmentHandlerTestCourse(t)

				secondCourse := models.Course{
					Title:  "Business English",
					Active: true,
				}

				if err := uadmin.Save(&secondCourse); err != nil {
					t.Fatalf("create second course: %v", err)
				}

				pkg := createEnrollmentHandlerTestPackage(t)

				first := models.Enrollment{
					StudentID: student.ID,
					CourseID:  firstCourse.ID,
					PackageID: pkg.ID,
				}

				if err := first.Create(); err != nil {
					t.Fatalf("create first enrollment: %v", err)
				}

				second := models.Enrollment{
					StudentID: student.ID,
					CourseID:  secondCourse.ID,
					PackageID: pkg.ID,
				}

				if err := second.Create(); err != nil {
					t.Fatalf("create second enrollment: %v", err)
				}

				return first, secondCourse
			},
			requestCourse: func(enrollment models.Enrollment, course models.Course) string {
				return strconv.FormatUint(uint64(course.ID), 10)
			},
			wantStatus:  "error",
			wantMessage: "The student already has an active enrollment for this course.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEnrollmentHandlerTestDB(t)

			enrollment, course := tt.setup(t)

			form := url.Values{}
			form.Set(
				"enrollment_id",
				strconv.FormatUint(uint64(enrollment.ID), 10),
			)
			form.Set(
				"course_id",
				tt.requestCourse(enrollment, course),
			)

			req := enrollmentHandlerFormRequest(
				http.MethodPost,
				"/enrollment/change-course/",
				form,
			)

			rec := httptest.NewRecorder()

			EnrollmentChangeCourseHandler(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf(
					"status = %d, want %d",
					rec.Code,
					http.StatusOK,
				)
			}

			var response map[string]interface{}

			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf(
					"decode response: %v\nbody=%s",
					err,
					rec.Body.String(),
				)
			}

			if response["status"] != tt.wantStatus {
				t.Fatalf(
					"response status = %v, want %s",
					response["status"],
					tt.wantStatus,
				)
			}

			if tt.wantMessage != "" {
				message, ok := response["message"].(string)

				if !ok {
					t.Fatalf(
						"response message = %v, want string",
						response["message"],
					)
				}

				if message != tt.wantMessage {
					t.Fatalf(
						"response message = %q, want %q",
						message,
						tt.wantMessage,
					)
				}
			}

			if tt.wantStatus == "ok" {
				var saved models.Enrollment

				if err := uadmin.GetDB().
					First(&saved, enrollment.ID).
					Error; err != nil {
					t.Fatalf(
						"failed to reload enrollment: %v",
						err,
					)
				}

				if saved.CourseID != course.ID {
					t.Fatalf(
						"CourseID = %d, want %d",
						saved.CourseID,
						course.ID,
					)
				}
			}
		})
	}
}

func TestEnrollmentChangeCourseHandlerMethodNotAllowed(t *testing.T) {
	setupEnrollmentHandlerTestDB(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/enrollment/change-course/",
		nil,
	)

	rec := httptest.NewRecorder()

	EnrollmentChangeCourseHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	var response map[string]interface{}

	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf(
			"decode response: %v\nbody=%s",
			err,
			rec.Body.String(),
		)
	}

	if response["status"] != "error" {
		t.Fatalf(
			"response status = %v, want error",
			response["status"],
		)
	}

	if response["message"] != "Method not allowed." {
		t.Fatalf(
			"response message = %v, want %q",
			response["message"],
			"Method not allowed.",
		)
	}
}

func TestEnrollmentChangeCourseError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "course required",
			err:  models.ErrEnrollmentChangeCourseRequired,
			want: "Select a course.",
		},
		{
			name: "enrollment not found",
			err:  models.ErrEnrollmentNotFound,
			want: "Enrollment not found.",
		},
		{
			name: "no credits",
			err:  models.ErrEnrollmentChangeCourseNoCredits,
			want: "This enrollment has no classes remaining.",
		},
		{
			name: "same course",
			err:  models.ErrEnrollmentChangeCourseSame,
			want: "The enrollment is already assigned to this course.",
		},
		{
			name: "course not found",
			err:  models.ErrEnrollmentCourseNotFound,
			want: "The selected course could not be found.",
		},
		{
			name: "inactive course",
			err:  models.ErrEnrollmentChangeCourseInactive,
			want: "The selected course is not available.",
		},
		{
			name: "duplicate enrollment",
			err:  models.ErrEnrollmentAlreadyExists,
			want: "The student already has an active enrollment for this course.",
		},
		{
			name: "unexpected error",
			err:  models.ErrEnrollmentChangeCourseFailed,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enrollmentChangeCourseError(tt.err)

			if got != tt.want {
				t.Fatalf(
					"error = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestEnrollmentDeactivateHandler(t *testing.T) {
	t.Run("deactivates active enrollment", func(t *testing.T) {
		setupEnrollmentHandlerTestDB(t)

		student := createEnrollmentHandlerTestStudent(t)
		course := createEnrollmentHandlerTestCourse(t)
		pkg := createEnrollmentHandlerTestPackage(t)

		enrollment := models.Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := enrollment.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		req := enrollmentHandlerFormRequest(
			http.MethodPost,
			"/enrollment/deactivate/",
			url.Values{
				"enrollment_id": {
					strconv.FormatUint(uint64(enrollment.ID), 10),
				},
			},
		)

		rec := httptest.NewRecorder()

		EnrollmentDeactivateHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusOK,
			)
		}

		var response map[string]interface{}

		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf(
				"decode response: %v\nbody=%s",
				err,
				rec.Body.String(),
			)
		}

		if response["status"] != "ok" {
			t.Fatalf(
				"response status = %v, want ok",
				response["status"],
			)
		}

		var saved models.Enrollment

		if err := uadmin.GetDB().
			First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if saved.Active {
			t.Fatal("saved Active = true, want false")
		}
	})

	t.Run("rejects invalid enrollment ID", func(t *testing.T) {
		setupEnrollmentHandlerTestDB(t)

		req := enrollmentHandlerFormRequest(
			http.MethodPost,
			"/enrollment/deactivate/",
			url.Values{
				"enrollment_id": {"invalid"},
			},
		)

		rec := httptest.NewRecorder()

		EnrollmentDeactivateHandler(rec, req)

		var response map[string]interface{}

		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf(
				"decode response: %v\nbody=%s",
				err,
				rec.Body.String(),
			)
		}

		if response["status"] != "error" {
			t.Fatalf(
				"response status = %v, want error",
				response["status"],
			)
		}

		if response["message"] != "Invalid enrollment ID." {
			t.Fatalf(
				"response message = %v, want Invalid enrollment ID.",
				response["message"],
			)
		}
	})

	t.Run("rejects already inactive enrollment", func(t *testing.T) {
		setupEnrollmentHandlerTestDB(t)

		student := createEnrollmentHandlerTestStudent(t)
		course := createEnrollmentHandlerTestCourse(t)
		pkg := createEnrollmentHandlerTestPackage(t)

		enrollment := models.Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := enrollment.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if err := uadmin.GetDB().
			Model(&models.Enrollment{}).
			Where("id = ?", enrollment.ID).
			Update("active", false).Error; err != nil {
			t.Fatalf("failed to deactivate test enrollment: %v", err)
		}

		req := enrollmentHandlerFormRequest(
			http.MethodPost,
			"/enrollment/deactivate/",
			url.Values{
				"enrollment_id": {
					strconv.FormatUint(uint64(enrollment.ID), 10),
				},
			},
		)

		rec := httptest.NewRecorder()

		EnrollmentDeactivateHandler(rec, req)

		var response map[string]interface{}

		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf(
				"decode response: %v\nbody=%s",
				err,
				rec.Body.String(),
			)
		}

		if response["status"] != "error" {
			t.Fatalf(
				"response status = %v, want error",
				response["status"],
			)
		}

		if response["message"] != "This enrollment is already inactive." {
			t.Fatalf(
				"response message = %v, want inactive message",
				response["message"],
			)
		}
	})
}
