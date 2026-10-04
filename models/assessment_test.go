package models

import (
	"testing"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

func setupAssessmentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	uadmin.ClearDB()
	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/assessment_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&uadmin.User{},
		&Level{},
		&Student{},
		&Course{},
		&Package{},
		&Enrollment{},
		&Class{},
		&Assessment{},
		&Homework{},
	); err != nil {
		t.Fatalf("failed to migrate assessment test database: %v", err)
	}

	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})

	return db
}

func createAssessmentTestClass(t *testing.T) Class {
	t.Helper()

	student := Student{
		FirstName: "Assessment",
		LastName:  "Test",
	}

	if err := uadmin.Save(&student); err != nil {
		t.Fatalf("failed to create student: %v", err)
	}

	course := Course{
		Title:  "Assessment Test Course",
		Active: true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("failed to create course: %v", err)
	}

	pkg := Package{
		Name:                   "Assessment Test Package",
		NumberOfClasses:        1,
		NumberOfFreeClasses:    0,
		ClassDurationInMinutes: 60,
		Price:                  100,
		ValidFrom:              timePtr(time.Now().Add(-24 * time.Hour)),
		ValidUntil:             timePtr(time.Now().Add(24 * time.Hour)),
		Active:                 true,
	}

	if err := uadmin.Save(&pkg); err != nil {
		t.Fatalf("failed to create package: %v", err)
	}

	enrollment := Enrollment{
		StudentID: student.ID,
		CourseID:  course.ID,
		PackageID: pkg.ID,
	}

	if err := enrollment.Create(); err != nil {
		t.Fatalf("failed to create enrollment: %v", err)
	}

	start := time.Now().Add(24 * time.Hour)
	end := start.Add(time.Hour)

	class := Class{
		ClassDate:    time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local),
		StartTime:    &start,
		EndTime:      &end,
		EnrollmentID: enrollment.ID,
		StudentID:    student.ID,
	}

	if err := uadmin.Save(&class); err != nil {
		t.Fatalf("failed to create class: %v", err)
	}

	return class
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func TestAssessmentClassUnique(t *testing.T) {
	db := setupAssessmentTestDB(t)

	class := createAssessmentTestClass(t)

	first := Assessment{
		Date:    time.Now(),
		ClassID: class.ID,
		Rating:  8,
	}

	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("failed to create first assessment: %v", err)
	}

	second := Assessment{
		Date:    time.Now(),
		ClassID: class.ID,
		Rating:  9,
	}

	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected duplicate assessment for the same class to fail")
	}
}

func TestHomeworkAssessmentUnique(t *testing.T) {
	db := setupAssessmentTestDB(t)

	class := createAssessmentTestClass(t)

	assessment := Assessment{
		Date:    time.Now(),
		ClassID: class.ID,
		Rating:  8,
	}

	if err := db.Create(&assessment).Error; err != nil {
		t.Fatalf("failed to create assessment: %v", err)
	}

	first := Homework{
		AssessmentID: assessment.ID,
		Title:        "First homework",
	}

	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("failed to create first homework: %v", err)
	}

	second := Homework{
		AssessmentID: assessment.ID,
		Title:        "Second homework",
	}

	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected duplicate homework for the same assessment to fail")
	}
}
