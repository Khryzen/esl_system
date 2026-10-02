package views

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func setupCourseMaterialHandlerTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/course_material_handler_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&models.Level{},
		&models.Course{},
		&models.Material{},
		&models.CourseMaterial{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
}

func createCourseMaterialHandlerTestCourse(t *testing.T) models.Course {
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

	return course
}

func TestCourseMaterialHandlerPOSTRejectsInvalidCourseID(t *testing.T) {
	setupCourseMaterialHandlerTestDB(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/course-material",
		strings.NewReader("courseID=not-a-number"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	CourseMaterialHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&models.CourseMaterial{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count course materials: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"course material count = %d, want 0",
			count,
		)
	}
}

func TestCourseMaterialHandlerPOSTRejectsMissingCourse(t *testing.T) {
	setupCourseMaterialHandlerTestDB(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/course-material",
		strings.NewReader("courseID=999999"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	CourseMaterialHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&models.CourseMaterial{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count course materials: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"course material count = %d, want 0",
			count,
		)
	}
}
