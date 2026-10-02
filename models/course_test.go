package models

import (
	"testing"

	"github.com/uadmin/uadmin"
)

func setupCourseTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/course_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&Course{},
		&Level{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
}

func createCourseTestLevel(t *testing.T, name string) Level {
	t.Helper()

	level := Level{
		Level: name,
	}

	uadmin.Save(&level)

	if level.ID == 0 {
		t.Fatal("level ID = 0, want persisted level")
	}

	return level
}

func TestCourseCreate(t *testing.T) {
	setupCourseTestDB(t)

	level := createCourseTestLevel(t, "Beginner")

	course := Course{
		Title:       "General English",
		Description: "General English course.",
		LevelID:     level.ID,
		Active:      true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("uadmin.Save() error = %v", err)
	}

	if course.ID == 0 {
		t.Fatal("course ID = 0, want persisted course")
	}

	var saved Course

	if err := uadmin.Get(&saved, "id = ?", course.ID); err != nil {
		t.Fatalf("failed to reload course: %v", err)
	}

	if saved.Title != course.Title {
		t.Fatalf(
			"Title = %q, want %q",
			saved.Title,
			course.Title,
		)
	}

	if saved.Description != course.Description {
		t.Fatalf(
			"Description = %q, want %q",
			saved.Description,
			course.Description,
		)
	}

	if saved.LevelID != level.ID {
		t.Fatalf(
			"LevelID = %d, want %d",
			saved.LevelID,
			level.ID,
		)
	}

	if !saved.Active {
		t.Fatal("Active = false, want true")
	}
}

func TestCourseUpdatePreservesID(t *testing.T) {
	setupCourseTestDB(t)

	firstLevel := createCourseTestLevel(t, "Beginner")
	secondLevel := createCourseTestLevel(t, "Intermediate")

	course := Course{
		Title:       "General English",
		Description: "Original description.",
		LevelID:     firstLevel.ID,
		Active:      true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("initial uadmin.Save() error = %v", err)
	}

	originalID := course.ID

	course.Title = "Business English"
	course.Description = "Updated description."
	course.LevelID = secondLevel.ID
	course.Active = false

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("update uadmin.Save() error = %v", err)
	}

	if course.ID != originalID {
		t.Fatalf(
			"course ID = %d, want unchanged ID %d",
			course.ID,
			originalID,
		)
	}

	var saved Course

	if err := uadmin.Get(&saved, "id = ?", originalID); err != nil {
		t.Fatalf("failed to reload course: %v", err)
	}

	if saved.ID != originalID {
		t.Fatalf(
			"saved ID = %d, want %d",
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

func TestCourseUpdateDoesNotCreateNewCourse(t *testing.T) {
	setupCourseTestDB(t)

	level := createCourseTestLevel(t, "Beginner")

	course := Course{
		Title:       "General English",
		Description: "Original description.",
		LevelID:     level.ID,
		Active:      true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("initial uadmin.Save() error = %v", err)
	}

	originalID := course.ID

	course.Title = "Updated English"

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("update uadmin.Save() error = %v", err)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&Course{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count courses: %v", err)
	}

	if count != 1 {
		t.Fatalf(
			"course count = %d, want 1",
			count,
		)
	}

	var saved Course

	if err := uadmin.Get(&saved, "id = ?", originalID); err != nil {
		t.Fatalf("failed to reload course: %v", err)
	}

	if saved.Title != "Updated English" {
		t.Fatalf(
			"Title = %q, want %q",
			saved.Title,
			"Updated English",
		)
	}
}

func TestCourseActiveCanBeChanged(t *testing.T) {
	setupCourseTestDB(t)

	level := createCourseTestLevel(t, "Beginner")

	course := Course{
		Title:   "General English",
		LevelID: level.ID,
		Active:  true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("initial uadmin.Save() error = %v", err)
	}

	course.Active = false

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("deactivation save error = %v", err)
	}

	var saved Course

	if err := uadmin.Get(&saved, "id = ?", course.ID); err != nil {
		t.Fatalf("failed to reload course: %v", err)
	}

	if saved.Active {
		t.Fatal("Active = true, want false")
	}

	course.Active = true

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("reactivation save error = %v", err)
	}

	if err := uadmin.Get(&saved, "id = ?", course.ID); err != nil {
		t.Fatalf("failed to reload reactivated course: %v", err)
	}

	if !saved.Active {
		t.Fatal("Active = false, want true")
	}
}
