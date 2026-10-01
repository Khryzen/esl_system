package models

import (
	"errors"
	"testing"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

func setupClassScheduleTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/class_test.db",
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
	); err != nil {
		t.Fatalf(
			"failed to migrate class test database: %v",
			err,
		)
	}

	t.Cleanup(func() {
		uadmin.ClearDB()
		uadmin.Database = nil
	})

	return db
}

func createClassScheduleTestEnrollment(t *testing.T, classes int) Enrollment {
	t.Helper()

	student := createEnrollmentTestStudent(t)
	course := createEnrollmentTestCourse(t)
	pkg := createEnrollmentTestPackage(t, classes, 0)

	enrollment := Enrollment{
		StudentID: student.ID,
		CourseID:  course.ID,
		PackageID: pkg.ID,
	}

	if err := enrollment.Create(); err != nil {
		t.Fatalf("failed to create test enrollment: %v", err)
	}

	return enrollment
}

func TestClassSchedule(t *testing.T) {
	t.Run("creates class and deducts one enrollment credit", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		if class.ID == 0 {
			t.Fatal("expected class ID to be assigned")
		}

		if class.StudentID != enrollment.StudentID {
			t.Fatalf(
				"StudentID = %d, want %d",
				class.StudentID,
				enrollment.StudentID,
			)
		}

		var saved Enrollment

		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if saved.ClassesRemaining != 4 {
			t.Fatalf(
				"ClassesRemaining = %d, want 4",
				saved.ClassesRemaining,
			)
		}

		if !saved.Active {
			t.Fatal("enrollment became inactive before all classes were consumed")
		}
	})

	t.Run("deactivates enrollment when last class is scheduled", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 1)

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		var saved Enrollment

		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if saved.ClassesRemaining != 0 {
			t.Fatalf(
				"ClassesRemaining = %d, want 0",
				saved.ClassesRemaining,
			)
		}

		if saved.Active {
			t.Fatal("enrollment Active = true, want false")
		}
	})

	t.Run("allows scheduling multiple classes until credits are exhausted", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 3)

		for i := 0; i < 3; i++ {
			class := Class{
				EnrollmentID: enrollment.ID,
			}

			if err := class.Schedule(); err != nil {
				t.Fatalf(
					"Schedule() error on class %d = %v",
					i+1,
					err,
				)
			}
		}

		var count int64

		if err := db.Model(&Class{}).
			Where("enrollment_id = ?", enrollment.ID).
			Count(&count).Error; err != nil {
			t.Fatalf("failed to count classes: %v", err)
		}

		if count != 3 {
			t.Fatalf(
				"class count = %d, want 3",
				count,
			)
		}

		var saved Enrollment

		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if saved.ClassesRemaining != 0 {
			t.Fatalf(
				"ClassesRemaining = %d, want 0",
				saved.ClassesRemaining,
			)
		}

		if saved.Active {
			t.Fatal("enrollment Active = true, want false")
		}
	})

	t.Run("rejects scheduling when no credits remain", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 1)

		first := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := first.Schedule(); err != nil {
			t.Fatalf("first Schedule() error = %v", err)
		}

		second := Class{
			EnrollmentID: enrollment.ID,
		}

		err := second.Schedule()

		if !errors.Is(err, ErrClassEnrollmentInactive) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrClassEnrollmentInactive,
			)
		}

		var count int64

		if err := db.Model(&Class{}).
			Where("enrollment_id = ?", enrollment.ID).
			Count(&count).Error; err != nil {
			t.Fatalf("failed to count classes: %v", err)
		}

		if count != 1 {
			t.Fatalf(
				"class count = %d, want 1",
				count,
			)
		}
	})

	t.Run("rejects inactive enrollment", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 5)

		if err := db.Model(&enrollment).
			Update("active", false).Error; err != nil {
			t.Fatalf("failed to deactivate enrollment: %v", err)
		}

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		err := class.Schedule()

		if !errors.Is(err, ErrClassEnrollmentInactive) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrClassEnrollmentInactive,
			)
		}
	})

	t.Run("rejects nonexistent enrollment", func(t *testing.T) {
		setupClassScheduleTestDB(t)

		class := Class{
			EnrollmentID: 99999,
		}

		err := class.Schedule()

		if !errors.Is(err, ErrClassEnrollmentNotFound) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrClassEnrollmentNotFound,
			)
		}
	})

	t.Run("rejects missing enrollment", func(t *testing.T) {
		setupClassScheduleTestDB(t)

		class := Class{}

		err := class.Schedule()

		if !errors.Is(err, ErrClassEnrollmentRequired) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrClassEnrollmentRequired,
			)
		}
	})

	t.Run("does not create class when scheduling fails", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 1)

		if err := db.Model(&enrollment).
			Update("active", false).Error; err != nil {
			t.Fatalf("failed to deactivate enrollment: %v", err)
		}

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		err := class.Schedule()

		if !errors.Is(err, ErrClassEnrollmentInactive) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrClassEnrollmentInactive,
			)
		}

		var count int64

		if err := db.Model(&Class{}).
			Where("enrollment_id = ?", enrollment.ID).
			Count(&count).Error; err != nil {
			t.Fatalf("failed to count classes: %v", err)
		}

		if count != 0 {
			t.Fatalf(
				"class count = %d, want 0",
				count,
			)
		}
	})

	t.Run("does not allow scheduling beyond available credits", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 1)

		first := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := first.Schedule(); err != nil {
			t.Fatalf("first Schedule() error = %v", err)
		}

		second := Class{
			EnrollmentID: enrollment.ID,
		}

		err := second.Schedule()

		if !errors.Is(err, ErrClassEnrollmentInactive) {
			t.Fatalf(
				"second Schedule() error = %v, want %v",
				err,
				ErrClassEnrollmentInactive,
			)
		}

		var updated Enrollment

		if err := db.First(&updated, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if updated.ClassesRemaining != 0 {
			t.Fatalf(
				"ClassesRemaining = %d, want 0",
				updated.ClassesRemaining,
			)
		}

		if updated.Active {
			t.Fatal("Active = true, want false")
		}
	})

	t.Run("stores enrollment course on class", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 1)

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		if class.CourseID != enrollment.CourseID {
			t.Fatalf(
				"CourseID = %d, want %d",
				class.CourseID,
				enrollment.CourseID,
			)
		}

		var saved Class

		if err := db.First(&saved, class.ID).Error; err != nil {
			t.Fatalf("failed to reload class: %v", err)
		}

		if saved.CourseID != enrollment.CourseID {
			t.Fatalf(
				"saved CourseID = %d, want %d",
				saved.CourseID,
				enrollment.CourseID,
			)
		}
	})
}

func TestClassCancel(t *testing.T) {
	t.Run("cancels class without refunding credit", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf("Cancel() error = %v", err)
		}

		var savedClass Class

		if err := db.First(&savedClass, class.ID).Error; err != nil {
			t.Fatalf("failed to reload class: %v", err)
		}

		if !savedClass.Cancelled {
			t.Fatal("Cancelled = false, want true")
		}

		if savedClass.CreditRefunded {
			t.Fatal("CreditRefunded = true, want false")
		}

		var savedEnrollment Enrollment

		if err := db.First(&savedEnrollment, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if savedEnrollment.ClassesRemaining != 4 {
			t.Fatalf(
				"ClassesRemaining = %d, want 4",
				savedEnrollment.ClassesRemaining,
			)
		}
	})

	t.Run("rejects cancelling an already cancelled class", func(t *testing.T) {
		setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf("first Cancel() error = %v", err)
		}

		err := class.Cancel()

		if !errors.Is(err, ErrClassAlreadyCancelled) {
			t.Fatalf(
				"second Cancel() error = %v, want %v",
				err,
				ErrClassAlreadyCancelled,
			)
		}
	})

	t.Run("rejects nonexistent class", func(t *testing.T) {
		setupClassScheduleTestDB(t)

		class := Class{
			Model: uadmin.Model{
				ID: 99999,
			},
		}

		err := class.Cancel()

		if !errors.Is(err, ErrClassNotFound) {
			t.Fatalf(
				"Cancel() error = %v, want %v",
				err,
				ErrClassNotFound,
			)
		}
	})
}
