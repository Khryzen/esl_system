package models

import (
	"testing"

	"github.com/uadmin/uadmin"
)

func setupCourseMaterialTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/course_material_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&Course{},
		&Level{},
		&Material{},
		&CourseMaterial{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
}

func createCourseMaterialTestCourse(
	t *testing.T,
	title string,
) Course {
	t.Helper()

	level := Level{
		Level: "Beginner",
	}

	if err := uadmin.Save(&level); err != nil {
		t.Fatalf("failed to create level: %v", err)
	}

	course := Course{
		Title:   title,
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

func createCourseMaterialTestMaterial(
	t *testing.T,
	name string,
) Material {
	t.Helper()

	material := Material{
		Name:   name,
		File:   "/uploads/" + name,
		Active: true,
	}

	if err := uadmin.Save(&material); err != nil {
		t.Fatalf("failed to create material: %v", err)
	}

	if material.ID == 0 {
		t.Fatal("material ID = 0, want persisted material")
	}

	return material
}

func TestCourseMaterialCreate(t *testing.T) {
	setupCourseMaterialTestDB(t)

	course := createCourseMaterialTestCourse(t, "General English")
	material := createCourseMaterialTestMaterial(t, "lesson-1.pdf")

	courseMaterial := CourseMaterial{
		CourseID:   course.ID,
		MaterialID: material.ID,
		Active:     true,
	}

	if err := uadmin.Save(&courseMaterial); err != nil {
		t.Fatalf("uadmin.Save() error = %v", err)
	}

	if courseMaterial.ID == 0 {
		t.Fatal("course material ID = 0, want persisted record")
	}

	var saved CourseMaterial

	if err := uadmin.Get(
		&saved,
		"id = ?",
		courseMaterial.ID,
	); err != nil {
		t.Fatalf("failed to reload course material: %v", err)
	}

	if saved.CourseID != course.ID {
		t.Fatalf(
			"CourseID = %d, want %d",
			saved.CourseID,
			course.ID,
		)
	}

	if saved.MaterialID != material.ID {
		t.Fatalf(
			"MaterialID = %d, want %d",
			saved.MaterialID,
			material.ID,
		)
	}

	if !saved.Active {
		t.Fatal("Active = false, want true")
	}
}

func TestCourseMaterialSaveIntegrity(t *testing.T) {
	setupCourseMaterialTestDB(t)

	firstCourse := createCourseMaterialTestCourse(
		t,
		"General English",
	)
	secondCourse := createCourseMaterialTestCourse(
		t,
		"Business English",
	)

	firstMaterial := createCourseMaterialTestMaterial(
		t,
		"lesson-1.pdf",
	)
	secondMaterial := createCourseMaterialTestMaterial(
		t,
		"lesson-2.pdf",
	)

	courseMaterial := CourseMaterial{
		CourseID:   firstCourse.ID,
		MaterialID: firstMaterial.ID,
		Active:     true,
	}

	if err := uadmin.Save(&courseMaterial); err != nil {
		t.Fatalf("initial uadmin.Save() error = %v", err)
	}

	courseMaterial.CourseID = secondCourse.ID
	courseMaterial.MaterialID = secondMaterial.ID
	courseMaterial.Active = false

	if err := uadmin.Save(&courseMaterial); err != nil {
		t.Fatalf("integrity-protected update error = %v", err)
	}

	var saved CourseMaterial

	if err := uadmin.Get(
		&saved,
		"id = ?",
		courseMaterial.ID,
	); err != nil {
		t.Fatalf("failed to reload course material: %v", err)
	}

	if saved.CourseID != firstCourse.ID {
		t.Fatalf(
			"CourseID = %d, want protected value %d",
			saved.CourseID,
			firstCourse.ID,
		)
	}

	if saved.MaterialID != firstMaterial.ID {
		t.Fatalf(
			"MaterialID = %d, want protected value %d",
			saved.MaterialID,
			firstMaterial.ID,
		)
	}

	if saved.Active {
		t.Fatal("Active = true, want false")
	}
}

func TestCourseMaterialActiveCanBeChanged(t *testing.T) {
	setupCourseMaterialTestDB(t)

	course := createCourseMaterialTestCourse(t, "General English")
	material := createCourseMaterialTestMaterial(t, "lesson-1.pdf")

	courseMaterial := CourseMaterial{
		CourseID:   course.ID,
		MaterialID: material.ID,
		Active:     true,
	}

	if err := uadmin.Save(&courseMaterial); err != nil {
		t.Fatalf("initial uadmin.Save() error = %v", err)
	}

	courseMaterial.Active = false

	if err := uadmin.Save(&courseMaterial); err != nil {
		t.Fatalf("deactivation save error = %v", err)
	}

	var saved CourseMaterial

	if err := uadmin.Get(
		&saved,
		"id = ?",
		courseMaterial.ID,
	); err != nil {
		t.Fatalf("failed to reload course material: %v", err)
	}

	if saved.Active {
		t.Fatal("Active = true, want false")
	}

	courseMaterial.Active = true

	if err := uadmin.Save(&courseMaterial); err != nil {
		t.Fatalf("reactivation save error = %v", err)
	}

	if err := uadmin.Get(
		&saved,
		"id = ?",
		courseMaterial.ID,
	); err != nil {
		t.Fatalf("failed to reload reactivated course material: %v", err)
	}

	if !saved.Active {
		t.Fatal("Active = false, want true")
	}
}
