package views

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func setupCourseHandlerTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/course_handler_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&models.Course{},
		&models.Level{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})
}

func createCourseHandlerTestLevel(t *testing.T, name string) models.Level {
	t.Helper()

	level := models.Level{
		Level: name,
	}

	if err := uadmin.Save(&level); err != nil {
		t.Fatalf("failed to create level: %v", err)
	}

	if level.ID == 0 {
		t.Fatal("level ID = 0, want persisted level")
	}

	return level
}

func courseHandlerFormRequest(
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

func TestCourseHandlerPOSTCreatesCourse(t *testing.T) {
	setupCourseHandlerTestDB(t)

	level := createCourseHandlerTestLevel(t, "Beginner")

	req := courseHandlerFormRequest(
		http.MethodPost,
		"/course",
		url.Values{
			"title":       {"General English"},
			"description": {"General English course."},
			"levelID":     {strconv.FormatUint(uint64(level.ID), 10)},
			"active":      {"true"},
		},
	)

	rec := httptest.NewRecorder()

	CourseHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d; body = %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	var courses []models.Course

	if err := uadmin.GetDB().Find(&courses).Error; err != nil {
		t.Fatalf("failed to query courses: %v", err)
	}

	if len(courses) != 1 {
		t.Fatalf("course count = %d, want 1", len(courses))
	}

	course := courses[0]

	if course.Title != "General English" {
		t.Fatalf(
			"Title = %q, want %q",
			course.Title,
			"General English",
		)
	}

	if course.Description != "General English course." {
		t.Fatalf(
			"Description = %q, want %q",
			course.Description,
			"General English course.",
		)
	}

	if course.LevelID != level.ID {
		t.Fatalf(
			"LevelID = %d, want %d",
			course.LevelID,
			level.ID,
		)
	}

	if !course.Active {
		t.Fatal("Active = false, want true")
	}
}

func TestCourseHandlerPUTUpdatesCourse(t *testing.T) {
	setupCourseHandlerTestDB(t)

	firstLevel := createCourseHandlerTestLevel(t, "Beginner")
	secondLevel := createCourseHandlerTestLevel(t, "Intermediate")

	course := models.Course{
		Title:       "General English",
		Description: "Original description.",
		LevelID:     firstLevel.ID,
		Active:      true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("failed to create course: %v", err)
	}

	originalID := course.ID

	req := courseHandlerFormRequest(
		http.MethodPut,
		"/course?id="+strconv.FormatUint(uint64(originalID), 10),
		url.Values{
			"title":       {"Business English"},
			"description": {"Updated description."},
			"levelID":     {strconv.FormatUint(uint64(secondLevel.ID), 10)},
			"active":      {"false"},
		},
	)

	rec := httptest.NewRecorder()

	CourseHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d; body = %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	var saved models.Course

	if err := uadmin.Get(
		&saved,
		"id = ?",
		originalID,
	); err != nil {
		t.Fatalf("failed to reload course: %v", err)
	}

	if saved.ID != originalID {
		t.Fatalf(
			"ID = %d, want %d",
			saved.ID,
			originalID,
		)
	}

	if saved.Title != "Business English" {
		t.Fatalf(
			"Title = %q, want %q",
			saved.Title,
			"Business English",
		)
	}

	if saved.Description != "Updated description." {
		t.Fatalf(
			"Description = %q, want %q",
			saved.Description,
			"Updated description.",
		)
	}

	if saved.LevelID != secondLevel.ID {
		t.Fatalf(
			"LevelID = %d, want %d",
			saved.LevelID,
			secondLevel.ID,
		)
	}

	if saved.Active {
		t.Fatal("Active = true, want false")
	}
}

func TestCourseHandlerPUTMissingCourse(t *testing.T) {
	setupCourseHandlerTestDB(t)

	level := createCourseHandlerTestLevel(t, "Beginner")

	req := courseHandlerFormRequest(
		http.MethodPut,
		"/course?id=999999",
		url.Values{
			"title":   {"Updated Course"},
			"levelID": {strconv.FormatUint(uint64(level.ID), 10)},
			"active":  {"true"},
		},
	)

	rec := httptest.NewRecorder()

	CourseHandler(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf(
			"status code = %d, want non-OK for missing course",
			rec.Code,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&models.Course{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count courses: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"course count = %d, want 0",
			count,
		)
	}
}

func TestCourseHandlerPOSTRejectsInvalidLevelID(t *testing.T) {
	setupCourseHandlerTestDB(t)

	req := courseHandlerFormRequest(
		http.MethodPost,
		"/course",
		url.Values{
			"title":       {"General English"},
			"description": {"General English course."},
			"levelID":     {"999999"},
			"active":      {"true"},
		},
	)

	rec := httptest.NewRecorder()

	CourseHandler(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf(
			"status code = %d, want non-OK for invalid level",
			rec.Code,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&models.Course{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count courses: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"course count = %d, want 0",
			count,
		)
	}
}

func TestCourseHandlerPOSTRejectsInvalidActive(t *testing.T) {
	setupCourseHandlerTestDB(t)

	level := createCourseHandlerTestLevel(t, "Beginner")

	req := courseHandlerFormRequest(
		http.MethodPost,
		"/course",
		url.Values{
			"title":       {"General English"},
			"description": {"General English course."},
			"levelID":     {strconv.FormatUint(uint64(level.ID), 10)},
			"active":      {"not-a-boolean"},
		},
	)

	rec := httptest.NewRecorder()

	CourseHandler(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf(
			"status code = %d, want non-OK for invalid active value",
			rec.Code,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&models.Course{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count courses: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"course count = %d, want 0",
			count,
		)
	}
}
