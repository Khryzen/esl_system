package views

import (
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func setupDashboardFeedbackTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/dashboard_feedback_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&uadmin.User{},
		&models.Student{},
		&models.Course{},
		&models.Package{},
		&models.Enrollment{},
		&models.Class{},
		&models.Assessment{},
		&models.Homework{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
}

func createDashboardFeedbackTestClass(
	t *testing.T,
	present bool,
) models.Class {
	t.Helper()

	student := models.Student{
		FirstName: "Test",
		LastName:  "Student",
	}

	if err := uadmin.Save(&student); err != nil {
		t.Fatalf("Save(student) error = %v", err)
	}

	course := models.Course{
		Title:  "Test Course",
		Active: true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("Save(course) error = %v", err)
	}

	pkg := models.Package{
		Name:                   "Test Package",
		NumberOfClasses:        10,
		NumberOfFreeClasses:    0,
		TotalClasses:           10,
		ClassDurationInMinutes: 30,
		Price:                  100,
		Active:                 true,
	}

	now := time.Now()
	later := now.Add(30 * 24 * time.Hour)

	pkg.ValidFrom = &now
	pkg.ValidUntil = &later

	if err := uadmin.Save(&pkg); err != nil {
		t.Fatalf("Save(package) error = %v", err)
	}

	enrollment := models.Enrollment{
		StudentID:       student.ID,
		CourseID:        course.ID,
		PackageID:       pkg.ID,
		TotalClasses:    10,
		ClassesRemaining: 9,
		ReferenceNumber: "TEST-FEEDBACK-001",
		Active:          true,
	}

	if err := uadmin.Save(&enrollment); err != nil {
		t.Fatalf("Save(enrollment) error = %v", err)
	}

	start := time.Now().Add(30 * time.Minute)
	end := start.Add(30 * time.Minute)

	class := models.Class{
		ClassDate:    time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local),
		StartTime:    &start,
		EndTime:      &end,
		EnrollmentID: enrollment.ID,
		CourseID:     course.ID,
		StudentID:    student.ID,
		Present:      present,
	}

	if err := uadmin.Save(&class); err != nil {
		t.Fatalf("Save(class) error = %v", err)
	}

	return class
}

func feedbackRequest(
	classID uint,
	rating string,
	homeworkTitle string,
) *httptest.ResponseRecorder {
	form := url.Values{}
	form.Set("class_id", strconv.FormatUint(uint64(classID), 10))
	form.Set("rating", rating)
	form.Set("grammar_corrections", "Grammar feedback")
	form.Set("recommendation", "Keep practicing")
	form.Set("homework", "Complete exercise 1")
	form.Set("remarks", "Good participation")
	form.Set("homework_title", homeworkTitle)

	req := httptest.NewRequest(
		"POST",
		"/dashboard",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	saveFeedback(rec, req)

	return rec
}

func TestSaveFeedbackCreatesAssessmentAndHomework(t *testing.T) {
	setupDashboardFeedbackTestDB(t)

	class := createDashboardFeedbackTestClass(t, true)

	rec := feedbackRequest(
		class.ID,
		"8.5",
		"Complete lesson 3",
	)

	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("saveFeedback() status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var assessments []models.Assessment

	uadmin.Filter(
		&assessments,
		"class_id = ?",
		class.ID,
	)

	if len(assessments) != 1 {
		t.Fatalf(
			"assessment count = %d, want 1",
			len(assessments),
		)
	}

	assessment := assessments[0]

	if assessment.Rating != 8.5 {
		t.Fatalf(
			"assessment.Rating = %v, want 8.5",
			assessment.Rating,
		)
	}

	if assessment.GrammarCorrections != "Grammar feedback" {
		t.Fatalf(
			"assessment.GrammarCorrections = %q, want %q",
			assessment.GrammarCorrections,
			"Grammar feedback",
		)
	}

	var homeworks []models.Homework

	uadmin.Filter(
		&homeworks,
		"assessment_id = ?",
		assessment.ID,
	)

	if len(homeworks) != 1 {
		t.Fatalf(
			"homework count = %d, want 1",
			len(homeworks),
		)
	}

	if homeworks[0].Title != "Complete lesson 3" {
		t.Fatalf(
			"homework.Title = %q, want %q",
			homeworks[0].Title,
			"Complete lesson 3",
		)
	}
}

func TestSaveFeedbackUpdatesExistingFeedback(t *testing.T) {
	setupDashboardFeedbackTestDB(t)

	class := createDashboardFeedbackTestClass(t, true)

	first := feedbackRequest(
		class.ID,
		"7.0",
		"First homework",
	)

	if first.Code < 200 || first.Code >= 300 {
		t.Fatalf(
			"first save status = %d, body = %s",
			first.Code,
			first.Body.String(),
		)
	}

	second := feedbackRequest(
		class.ID,
		"9.0",
		"Updated homework",
	)

	if second.Code < 200 || second.Code >= 300 {
		t.Fatalf(
			"second save status = %d, body = %s",
			second.Code,
			second.Body.String(),
		)
	}

	var assessments []models.Assessment

	uadmin.Filter(
		&assessments,
		"class_id = ?",
		class.ID,
	)

	if len(assessments) != 1 {
		t.Fatalf(
			"assessment count after update = %d, want 1",
			len(assessments),
		)
	}

	assessment := assessments[0]

	if assessment.Rating != 9.0 {
		t.Fatalf(
			"assessment.Rating = %v, want 9.0",
			assessment.Rating,
		)
	}

	var homeworks []models.Homework

	uadmin.Filter(
		&homeworks,
		"assessment_id = ?",
		assessment.ID,
	)

	if len(homeworks) != 1 {
		t.Fatalf(
			"homework count after update = %d, want 1",
			len(homeworks),
		)
	}

	if homeworks[0].Title != "Updated homework" {
		t.Fatalf(
			"homework.Title = %q, want %q",
			homeworks[0].Title,
			"Updated homework",
		)
	}
}

func TestSaveFeedbackRejectsAbsentClass(t *testing.T) {
	setupDashboardFeedbackTestDB(t)

	class := createDashboardFeedbackTestClass(t, false)

	rec := feedbackRequest(
		class.ID,
		"8.0",
		"Homework",
	)

	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf(
			"saveFeedback() status = %d, body = %s",
			rec.Code,
			rec.Body.String(),
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"Feedback can only be given for a class marked present.",
	) {
		t.Fatalf(
			"response = %q, want present-class validation error",
			rec.Body.String(),
		)
	}

	var assessments []models.Assessment

	uadmin.Filter(
		&assessments,
		"class_id = ?",
		class.ID,
	)

	if len(assessments) != 0 {
		t.Fatalf(
			"assessment count = %d, want 0",
			len(assessments),
		)
	}
}

func TestSaveFeedbackRejectsInvalidRating(t *testing.T) {
	setupDashboardFeedbackTestDB(t)

	class := createDashboardFeedbackTestClass(t, true)

	rec := feedbackRequest(
		class.ID,
		"11",
		"Homework",
	)

	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf(
			"saveFeedback() status = %d, body = %s",
			rec.Code,
			rec.Body.String(),
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"Enter a rating between 0 and 10.",
	) {
		t.Fatalf(
			"response = %q, want rating validation error",
			rec.Body.String(),
		)
	}

	var assessments []models.Assessment

	uadmin.Filter(
		&assessments,
		"class_id = ?",
		class.ID,
	)

	if len(assessments) != 0 {
		t.Fatalf(
			"assessment count = %d, want 0",
			len(assessments),
		)
	}
}