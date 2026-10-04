package views

import (
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

	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})
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
		StudentID:        student.ID,
		CourseID:         course.ID,
		PackageID:        pkg.ID,
		TotalClasses:     10,
		ClassesRemaining: 9,
		ReferenceNumber:  "TEST-FEEDBACK-001",
		Active:           true,
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
