package models

import (
	"errors"
	"testing"
	"time"

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
		t.Fatalf("failed to migrate class test database: %v", err)
	}

	t.Cleanup(func() {
		uadmin.ClearDB()
		uadmin.Database = nil
	})

	return db
}

func createClassScheduleTestEnrollment(
	t *testing.T,
	classes int,
) Enrollment {
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

	var saved Enrollment
	if err := uadmin.GetDB().
		First(&saved, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload test enrollment: %v", err)
	}

	return saved
}

func newTestClass(
	enrollmentID uint,
	daysFromToday int,
	hour int,
) Class {
	now := time.Now().In(time.Local)

	classDate := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+daysFromToday,
		0,
		0,
		0,
		0,
		now.Location(),
	)

	startTime := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+daysFromToday,
		hour,
		0,
		0,
		0,
		now.Location(),
	)

	return Class{
		ClassDate:    classDate,
		StartTime:    &startTime,
		EnrollmentID: enrollmentID,
	}
}

func newTestClassAt(
	enrollmentID uint,
	daysFromToday int,
	hour int,
	minute int,
) Class {
	now := time.Now().In(time.Local)

	classDate := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+daysFromToday,
		0,
		0,
		0,
		0,
		now.Location(),
	)

	startTime := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+daysFromToday,
		hour,
		minute,
		0,
		0,
		now.Location(),
	)

	return Class{
		ClassDate:    classDate,
		StartTime:    &startTime,
		EnrollmentID: enrollmentID,
	}
}

func TestClassSchedule(t *testing.T) {
	t.Run("creates class and deducts one enrollment credit", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(enrollment.ID, 1, 10)

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

		class := newTestClass(enrollment.ID, 1, 10)

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
			class := newTestClass(
				enrollment.ID,
				1,
				9+i,
			)

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

		first := newTestClass(enrollment.ID, 1, 10)

		if err := first.Schedule(); err != nil {
			t.Fatalf("first Schedule() error = %v", err)
		}

		second := newTestClass(enrollment.ID, 1, 11)

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
		setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		if err := uadmin.GetDB().
			Model(&enrollment).
			Update("active", false).Error; err != nil {
			t.Fatalf("failed to deactivate enrollment: %v", err)
		}

		class := newTestClass(enrollment.ID, 1, 10)

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

		class := newTestClass(99999, 1, 10)

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

		class := newTestClass(enrollment.ID, 1, 10)

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

		first := newTestClass(enrollment.ID, 1, 10)

		if err := first.Schedule(); err != nil {
			t.Fatalf("first Schedule() error = %v", err)
		}

		second := newTestClass(enrollment.ID, 1, 11)

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

		class := newTestClass(enrollment.ID, 1, 10)

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

	t.Run("rejects invalid package class duration", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		if err := db.Model(&Package{}).
			Where("id = ?", enrollment.PackageID).
			Update("class_duration_in_minutes", 0).Error; err != nil {
			t.Fatalf(
				"failed to update package duration: %v",
				err,
			)
		}

		class := newTestClass(enrollment.ID, 1, 10)

		err := class.Schedule()

		if !errors.Is(err, ErrPackageInvalidClassDuration) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrPackageInvalidClassDuration,
			)
		}

		var count int64
		if err := db.Model(&Class{}).
			Where("enrollment_id = ?", enrollment.ID).
			Count(&count).Error; err != nil {
			t.Fatalf(
				"failed to count classes: %v",
				err,
			)
		}

		if count != 0 {
			t.Fatalf(
				"class count = %d, want 0",
				count,
			)
		}
	})

	t.Run("allows adjacent classes", func(t *testing.T) {
		setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		first := newTestClass(enrollment.ID, 1, 9)

		if err := first.Schedule(); err != nil {
			t.Fatalf(
				"first Schedule() error = %v",
				err,
			)
		}

		second := newTestClass(enrollment.ID, 1, 10)

		if err := second.Schedule(); err != nil {
			t.Fatalf(
				"second Schedule() error = %v",
				err,
			)
		}
	})

	t.Run("rejects overlapping classes", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		first := newTestClassAt(
			enrollment.ID,
			1,
			9,
			0,
		)

		if err := first.Schedule(); err != nil {
			t.Fatalf(
				"first Schedule() error = %v",
				err,
			)
		}

		second := newTestClassAt(
			enrollment.ID,
			1,
			9,
			30,
		)

		err := second.Schedule()

		if !errors.Is(err, ErrClassScheduleConflict) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrClassScheduleConflict,
			)
		}

		var count int64
		if err := db.Model(&Class{}).
			Where("enrollment_id = ?", enrollment.ID).
			Count(&count).Error; err != nil {
			t.Fatalf(
				"failed to count classes: %v",
				err,
			)
		}

		if count != 1 {
			t.Fatalf(
				"class count = %d, want 1",
				count,
			)
		}
	})

	t.Run("allows scheduling when the conflicting class is cancelled", func(t *testing.T) {
		setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		first := newTestClass(enrollment.ID, 1, 9)

		if err := first.Schedule(); err != nil {
			t.Fatalf(
				"first Schedule() error = %v",
				err,
			)
		}

		if err := first.Cancel(); err != nil {
			t.Fatalf(
				"Cancel() error = %v",
				err,
			)
		}

		second := newTestClass(enrollment.ID, 1, 9)

		if err := second.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v, want cancelled class to stop blocking",
				err,
			)
		}
	})

	t.Run("calculates end time from package duration", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(enrollment.ID, 1, 10)

		wrongEnd := time.Date(
			time.Now().In(time.Local).Year(),
			time.Now().In(time.Local).Month(),
			time.Now().In(time.Local).Day()+1,
			18,
			30,
			0,
			0,
			time.Local,
		)

		class.EndTime = &wrongEnd

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		if class.StartTime == nil {
			t.Fatal("StartTime is nil")
		}

		if class.EndTime == nil {
			t.Fatal("EndTime is nil")
		}

		expectedEnd := class.StartTime.Add(60 * time.Minute)

		if !class.EndTime.Equal(expectedEnd) {
			t.Fatalf(
				"EndTime = %v, want %v",
				class.EndTime,
				expectedEnd,
			)
		}

		var saved Class
		if err := db.First(&saved, class.ID).Error; err != nil {
			t.Fatalf(
				"failed to reload class: %v",
				err,
			)
		}

		if saved.EndTime == nil {
			t.Fatal("saved EndTime is nil")
		}

		if !saved.EndTime.Equal(expectedEnd) {
			t.Fatalf(
				"saved EndTime = %v, want %v",
				saved.EndTime,
				expectedEnd,
			)
		}
	})

	t.Run("rejects class with missing date", func(t *testing.T) {
		setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		now := time.Now().In(time.Local)

		startTime := time.Date(
			now.Year(),
			now.Month(),
			now.Day()+1,
			10,
			0,
			0,
			0,
			now.Location(),
		)

		class := Class{
			EnrollmentID: enrollment.ID,
			StartTime:    &startTime,
		}

		err := class.Schedule()

		if !errors.Is(err, ErrClassDateRequired) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrClassDateRequired,
			)
		}
	})

	t.Run("rejects class scheduled in the past", func(t *testing.T) {
		setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(
			enrollment.ID,
			-1,
			10,
		)

		err := class.Schedule()

		if !errors.Is(err, ErrClassDateInPast) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrClassDateInPast,
			)
		}
	})

	t.Run("rejects today's class when start time has passed", func(t *testing.T) {
		setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		now := time.Now().In(time.Local)
		pastHour := now.Add(-2 * time.Hour)

		classDate := time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			0,
			0,
			0,
			0,
			now.Location(),
		)

		startTime := time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			pastHour.Hour(),
			pastHour.Minute(),
			0,
			0,
			now.Location(),
		)

		class := Class{
			ClassDate:    classDate,
			StartTime:    &startTime,
			EnrollmentID: enrollment.ID,
		}

		err := class.Schedule()

		if !errors.Is(err, ErrClassStartTimeInPast) {
			t.Fatalf(
				"Schedule() error = %v, want %v",
				err,
				ErrClassStartTimeInPast,
			)
		}
	})

	t.Run("rejects already scheduled class", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"first Schedule() error = %v",
				err,
			)
		}

		before := class

		err := class.Schedule()

		if !errors.Is(err, ErrClassAlreadyScheduled) {
			t.Fatalf(
				"second Schedule() error = %v, want %v",
				err,
				ErrClassAlreadyScheduled,
			)
		}

		var count int64
		if err := db.Model(&Class{}).
			Where("enrollment_id = ?", enrollment.ID).
			Count(&count).Error; err != nil {
			t.Fatalf(
				"failed to count classes: %v",
				err,
			)
		}

		if count != 1 {
			t.Fatalf(
				"class count = %d, want 1",
				count,
			)
		}

		var saved Enrollment
		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if saved.ClassesRemaining != 4 {
			t.Fatalf(
				"ClassesRemaining = %d, want 4",
				saved.ClassesRemaining,
			)
		}

		if class.ID != before.ID {
			t.Fatalf(
				"class ID changed from %d to %d",
				before.ID,
				class.ID,
			)
		}
	})
}

func TestClassCancel(t *testing.T) {
	t.Run("cancels class without refunding credit", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf(
				"Cancel() error = %v",
				err,
			)
		}

		var savedClass Class
		if err := db.First(&savedClass, class.ID).Error; err != nil {
			t.Fatalf(
				"failed to reload class: %v",
				err,
			)
		}

		if !savedClass.Cancelled {
			t.Fatal("Cancelled = false, want true")
		}

		if savedClass.CreditRefunded {
			t.Fatal("CreditRefunded = true, want false")
		}

		var savedEnrollment Enrollment
		if err := db.First(
			&savedEnrollment,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
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

		class := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf(
				"first Cancel() error = %v",
				err,
			)
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

func TestClassRefundCredit(t *testing.T) {
	t.Run("refunds one credit from a cancelled class", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf(
				"Cancel() error = %v",
				err,
			)
		}

		if err := class.RefundCredit(); err != nil {
			t.Fatalf(
				"RefundCredit() error = %v",
				err,
			)
		}

		var savedClass Class
		if err := db.First(
			&savedClass,
			class.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload class: %v",
				err,
			)
		}

		if !savedClass.Cancelled {
			t.Fatal("Cancelled = false, want true")
		}

		if !savedClass.CreditRefunded {
			t.Fatal("CreditRefunded = false, want true")
		}

		var savedEnrollment Enrollment
		if err := db.First(
			&savedEnrollment,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if savedEnrollment.ClassesRemaining != 5 {
			t.Fatalf(
				"ClassesRemaining = %d, want 5",
				savedEnrollment.ClassesRemaining,
			)
		}

		if !savedEnrollment.Active {
			t.Fatal("Active = false, want true")
		}
	})

	t.Run("reactivates exhausted enrollment when credit is refunded", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 1)

		class := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf(
				"Cancel() error = %v",
				err,
			)
		}

		var exhausted Enrollment
		if err := db.First(
			&exhausted,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if exhausted.ClassesRemaining != 0 {
			t.Fatalf(
				"ClassesRemaining before refund = %d, want 0",
				exhausted.ClassesRemaining,
			)
		}

		if exhausted.Active {
			t.Fatal("Active before refund = true, want false")
		}

		if err := class.RefundCredit(); err != nil {
			t.Fatalf(
				"RefundCredit() error = %v",
				err,
			)
		}

		var saved Enrollment
		if err := db.First(
			&saved,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if saved.ClassesRemaining != 1 {
			t.Fatalf(
				"ClassesRemaining = %d, want 1",
				saved.ClassesRemaining,
			)
		}

		if !saved.Active {
			t.Fatal("Active = false, want true")
		}
	})

	t.Run("rejects refunding a class that was not cancelled", func(t *testing.T) {
		setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		err := class.RefundCredit()

		if !errors.Is(err, ErrClassNotCancelled) {
			t.Fatalf(
				"RefundCredit() error = %v, want %v",
				err,
				ErrClassNotCancelled,
			)
		}
	})

	t.Run("rejects refunding the same class twice", func(t *testing.T) {
		setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf(
				"Cancel() error = %v",
				err,
			)
		}

		if err := class.RefundCredit(); err != nil {
			t.Fatalf(
				"first RefundCredit() error = %v",
				err,
			)
		}

		err := class.RefundCredit()

		if !errors.Is(err, ErrClassCreditAlreadyRefunded) {
			t.Fatalf(
				"second RefundCredit() error = %v, want %v",
				err,
				ErrClassCreditAlreadyRefunded,
			)
		}
	})

	t.Run("preserves enrollment credit balance when rescheduling the final scheduled credit", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 2)

		first := newTestClass(enrollment.ID, 1, 10)
		if err := first.Schedule(); err != nil {
			t.Fatalf("first Schedule() error = %v", err)
		}

		var before Enrollment
		if err := db.First(&before, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if before.ClassesRemaining != 1 {
			t.Fatalf(
				"ClassesRemaining before reschedule = %d, want 1",
				before.ClassesRemaining,
			)
		}

		if !before.Active {
			t.Fatal("enrollment became inactive before final scheduled credit was rescheduled")
		}

		replacement := newTestClass(0, 2, 10)
		if err := first.Reschedule(&replacement); err != nil {
			t.Fatalf("Reschedule() error = %v", err)
		}

		var after Enrollment
		if err := db.First(&after, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if after.ClassesRemaining != before.ClassesRemaining {
			t.Fatalf(
				"ClassesRemaining after reschedule = %d, want %d",
				after.ClassesRemaining,
				before.ClassesRemaining,
			)
		}

		if after.Active != before.Active {
			t.Fatalf(
				"enrollment Active after reschedule = %t, want %t",
				after.Active,
				before.Active,
			)
		}

		var original Class
		if err := db.First(&original, first.ID).Error; err != nil {
			t.Fatalf("failed to reload original class: %v", err)
		}

		if !original.Cancelled {
			t.Fatal("original Cancelled = false, want true")
		}

		if !original.CreditRefunded {
			t.Fatal("original CreditRefunded = false, want true")
		}

		var savedReplacement Class
		if err := db.First(&savedReplacement, replacement.ID).Error; err != nil {
			t.Fatalf("failed to reload replacement class: %v", err)
		}

		if savedReplacement.Cancelled {
			t.Fatal("replacement Cancelled = true, want false")
		}

		if savedReplacement.CreditRefunded {
			t.Fatal("replacement CreditRefunded = true, want false")
		}
	})

	t.Run("rejects nonexistent class", func(t *testing.T) {
		setupClassScheduleTestDB(t)

		class := Class{
			Model: uadmin.Model{
				ID: 99999,
			},
		}

		err := class.RefundCredit()

		if !errors.Is(err, ErrClassNotFound) {
			t.Fatalf(
				"RefundCredit() error = %v, want %v",
				err,
				ErrClassNotFound,
			)
		}
	})

	t.Run("rejects refund when enrollment does not exist", func(t *testing.T) {
		setupClassScheduleTestDB(t)

		class := newTestClass(
			99999,
			1,
			10,
		)

		class.Cancelled = true

		if err := uadmin.GetDB().Create(&class).Error; err != nil {
			t.Fatalf(
				"failed to create test class: %v",
				err,
			)
		}

		err := class.RefundCredit()

		if !errors.Is(err, ErrClassEnrollmentNotFound) {
			t.Fatalf(
				"RefundCredit() error = %v, want %v",
				err,
				ErrClassEnrollmentNotFound,
			)
		}
	})

	t.Run("rejects refund when enrollment already has all credits", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := Class{
			ClassDate: time.Now().
				In(time.Local).
				AddDate(0, 0, 1),
			EnrollmentID: enrollment.ID,
			Cancelled:    true,
		}

		if err := db.Create(&class).Error; err != nil {
			t.Fatalf(
				"failed to create cancelled test class: %v",
				err,
			)
		}

		err := class.RefundCredit()

		if !errors.Is(err, ErrClassCreditsAlreadyFull) {
			t.Fatalf(
				"RefundCredit() error = %v, want %v",
				err,
				ErrClassCreditsAlreadyFull,
			)
		}

		var saved Enrollment
		if err := db.First(
			&saved,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if saved.ClassesRemaining != saved.TotalClasses {
			t.Fatalf(
				"ClassesRemaining = %d, want %d",
				saved.ClassesRemaining,
				saved.TotalClasses,
			)
		}

		if !saved.Active {
			t.Fatal("Active = false, want true")
		}

		var savedClass Class
		if err := db.First(
			&savedClass,
			class.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload class: %v",
				err,
			)
		}

		if savedClass.CreditRefunded {
			t.Fatal("CreditRefunded = true, want false")
		}
	})
}

func TestClassReschedule(t *testing.T) {
	t.Run("cancels original class and creates replacement", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		original := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := original.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		var originalBefore Class
		if err := db.First(
			&originalBefore,
			original.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload original class: %v",
				err,
			)
		}

		replacement := newTestClass(
			0,
			2,
			10,
		)

		if err := original.Reschedule(&replacement); err != nil {
			t.Fatalf(
				"Reschedule() error = %v",
				err,
			)
		}

		var savedOriginal Class
		if err := db.First(
			&savedOriginal,
			original.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload original class: %v",
				err,
			)
		}

		if !savedOriginal.Cancelled {
			t.Fatal("original Cancelled = false, want true")
		}

		if !savedOriginal.CreditRefunded {
			t.Fatal("original CreditRefunded = false, want true")
		}

		var savedReplacement Class
		if err := db.First(
			&savedReplacement,
			replacement.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload replacement class: %v",
				err,
			)
		}

		if savedReplacement.ID == savedOriginal.ID {
			t.Fatal("replacement reused original class ID")
		}

		if savedReplacement.Cancelled {
			t.Fatal("replacement Cancelled = true, want false")
		}

		if savedReplacement.CreditRefunded {
			t.Fatal("replacement CreditRefunded = true, want false")
		}

		if savedReplacement.EnrollmentID != originalBefore.EnrollmentID {
			t.Fatalf(
				"replacement EnrollmentID = %d, want %d",
				savedReplacement.EnrollmentID,
				originalBefore.EnrollmentID,
			)
		}

		if savedReplacement.StudentID != originalBefore.StudentID {
			t.Fatalf(
				"replacement StudentID = %d, want %d",
				savedReplacement.StudentID,
				originalBefore.StudentID,
			)
		}

		if savedReplacement.CourseID != originalBefore.CourseID {
			t.Fatalf(
				"replacement CourseID = %d, want %d",
				savedReplacement.CourseID,
				originalBefore.CourseID,
			)
		}

		if savedReplacement.RescheduledFromID != original.ID {
			t.Fatalf(
				"replacement RescheduledFromID = %d, want %d",
				savedReplacement.RescheduledFromID,
				original.ID,
			)
		}
	})

	t.Run("does not change enrollment credit balance", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		original := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := original.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		var before Enrollment
		if err := db.First(
			&before,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		replacement := newTestClass(
			0,
			2,
			11,
		)

		if err := original.Reschedule(&replacement); err != nil {
			t.Fatalf(
				"Reschedule() error = %v",
				err,
			)
		}

		var after Enrollment
		if err := db.First(
			&after,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if after.ClassesRemaining != before.ClassesRemaining {
			t.Fatalf(
				"ClassesRemaining = %d, want %d",
				after.ClassesRemaining,
				before.ClassesRemaining,
			)
		}

		if after.Active != before.Active {
			t.Fatalf(
				"Active = %t, want %t",
				after.Active,
				before.Active,
			)
		}
	})

	t.Run("rejects replacement that already exists", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		original := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := original.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		replacement := newTestClass(
			enrollment.ID,
			1,
			11,
		)

		replacement.ID = original.ID

		err := original.Reschedule(&replacement)

		if !errors.Is(err, ErrClassReplacementAlreadyExists) {
			t.Fatalf(
				"Reschedule() error = %v, want %v",
				err,
				ErrClassReplacementAlreadyExists,
			)
		}

		var savedOriginal Class
		if err := db.First(
			&savedOriginal,
			original.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload original class: %v",
				err,
			)
		}

		if savedOriginal.Cancelled {
			t.Fatal("original Cancelled = true, want false")
		}

		if savedOriginal.CreditRefunded {
			t.Fatal("original CreditRefunded = true, want false")
		}

		var classCount int64
		if err := db.Model(&Class{}).
			Where("enrollment_id = ?", enrollment.ID).
			Count(&classCount).Error; err != nil {
			t.Fatalf(
				"failed to count enrollment classes: %v",
				err,
			)
		}

		if classCount != 1 {
			t.Fatalf(
				"class count = %d, want 1",
				classCount,
			)
		}
	})

	t.Run("rejects cancelling a class with recorded attendance", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		if err := db.Model(&class).
			Update("present", true).Error; err != nil {
			t.Fatalf(
				"failed to record attendance: %v",
				err,
			)
		}

		err := class.Cancel()

		if !errors.Is(err, ErrClassAttendanceRecorded) {
			t.Fatalf(
				"Cancel() error = %v, want %v",
				err,
				ErrClassAttendanceRecorded,
			)
		}

		var saved Class
		if err := db.First(
			&saved,
			class.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload class: %v",
				err,
			)
		}

		if saved.Cancelled {
			t.Fatal("Cancelled = true, want false")
		}

		if !saved.Present {
			t.Fatal("Present = false, want true")
		}
	})

	t.Run("rejects cancelling a class with recorded absence", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		class := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := class.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		if err := db.Model(&class).
			Update("absent", true).Error; err != nil {
			t.Fatalf(
				"failed to record absence: %v",
				err,
			)
		}

		err := class.Cancel()

		if !errors.Is(err, ErrClassAttendanceRecorded) {
			t.Fatalf(
				"Cancel() error = %v, want %v",
				err,
				ErrClassAttendanceRecorded,
			)
		}

		var saved Class
		if err := db.First(
			&saved,
			class.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload class: %v",
				err,
			)
		}

		if saved.Cancelled {
			t.Fatal("Cancelled = true, want false")
		}

		if !saved.Absent {
			t.Fatal("Absent = false, want true")
		}
	})

	t.Run("rejects rescheduling when enrollment course has changed", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		originalCourseID := enrollment.CourseID

		secondCourse := Course{
			Title:  "Business English",
			Active: true,
		}

		if err := uadmin.Save(&secondCourse); err != nil {
			t.Fatalf(
				"failed to create second test course: %v",
				err,
			)
		}

		original := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := original.Schedule(); err != nil {
			t.Fatalf(
				"Schedule() error = %v",
				err,
			)
		}

		if original.CourseID != originalCourseID {
			t.Fatalf(
				"original CourseID = %d, want %d",
				original.CourseID,
				originalCourseID,
			)
		}

		if err := enrollment.ChangeCourse(secondCourse.ID); err != nil {
			t.Fatalf(
				"ChangeCourse() error = %v",
				err,
			)
		}

		replacement := newTestClass(
			0,
			2,
			10,
		)

		err := original.Reschedule(&replacement)

		if !errors.Is(err, ErrClassRescheduleCourseChanged) {
			t.Fatalf(
				"Reschedule() error = %v, want %v",
				err,
				ErrClassRescheduleCourseChanged,
			)
		}

		var savedOriginal Class
		if err := db.First(
			&savedOriginal,
			original.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload original class: %v",
				err,
			)
		}

		if savedOriginal.Cancelled {
			t.Fatal("original Cancelled = true, want false")
		}

		if savedOriginal.CreditRefunded {
			t.Fatal("original CreditRefunded = true, want false")
		}

		if savedOriginal.CourseID != originalCourseID {
			t.Fatalf(
				"original CourseID = %d, want %d",
				savedOriginal.CourseID,
				originalCourseID,
			)
		}

		var savedEnrollment Enrollment
		if err := db.First(
			&savedEnrollment,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if savedEnrollment.CourseID != secondCourse.ID {
			t.Fatalf(
				"enrollment CourseID = %d, want %d",
				savedEnrollment.CourseID,
				secondCourse.ID,
			)
		}

		var count int64
		if err := db.Model(&Class{}).
			Where("enrollment_id = ?", enrollment.ID).
			Count(&count).Error; err != nil {
			t.Fatalf(
				"failed to count classes: %v",
				err,
			)
		}

		if count != 1 {
			t.Fatalf(
				"class count = %d, want 1",
				count,
			)
		}
	})

	t.Run("rolls back when replacement conflicts with another class", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 5)

		original := newTestClassAt(
			enrollment.ID,
			1,
			9,
			0,
		)

		if err := original.Schedule(); err != nil {
			t.Fatalf(
				"original Schedule() error = %v",
				err,
			)
		}

		conflicting := newTestClassAt(
			enrollment.ID,
			2,
			9,
			0,
		)

		if err := conflicting.Schedule(); err != nil {
			t.Fatalf(
				"conflicting Schedule() error = %v",
				err,
			)
		}

		var before Enrollment
		if err := db.First(
			&before,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		replacement := newTestClassAt(
			0,
			2,
			9,
			30,
		)

		err := original.Reschedule(&replacement)

		if !errors.Is(err, ErrClassScheduleConflict) {
			t.Fatalf(
				"Reschedule() error = %v, want %v",
				err,
				ErrClassScheduleConflict,
			)
		}

		var savedOriginal Class
		if err := db.First(
			&savedOriginal,
			original.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload original class: %v",
				err,
			)
		}

		if savedOriginal.Cancelled {
			t.Fatal("original Cancelled = true, want false")
		}

		if savedOriginal.CreditRefunded {
			t.Fatal("original CreditRefunded = true, want false")
		}

		var replacementCount int64
		if err := db.Model(&Class{}).
			Where(
				"rescheduled_from_id = ?",
				original.ID,
			).
			Count(&replacementCount).Error; err != nil {
			t.Fatalf(
				"failed to count replacement classes: %v",
				err,
			)
		}

		if replacementCount != 0 {
			t.Fatalf(
				"replacement count = %d, want 0",
				replacementCount,
			)
		}

		var after Enrollment
		if err := db.First(
			&after,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if after.ClassesRemaining != before.ClassesRemaining {
			t.Fatalf(
				"ClassesRemaining = %d, want %d",
				after.ClassesRemaining,
				before.ClassesRemaining,
			)
		}

		if after.Active != before.Active {
			t.Fatalf(
				"Active = %t, want %t",
				after.Active,
				before.Active,
			)
		}
	})

	t.Run("preserves remaining credit when rescheduling a class", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)
		enrollment := createClassScheduleTestEnrollment(t, 2)

		first := newTestClass(
			enrollment.ID,
			1,
			10,
		)

		if err := first.Schedule(); err != nil {
			t.Fatalf(
				"first Schedule() error = %v",
				err,
			)
		}

		var before Enrollment
		if err := db.First(
			&before,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if before.ClassesRemaining != 1 {
			t.Fatalf(
				"ClassesRemaining before reschedule = %d, want 1",
				before.ClassesRemaining,
			)
		}

		if !before.Active {
			t.Fatal(
				"enrollment became inactive before reschedule",
			)
		}

		replacement := newTestClass(
			0,
			2,
			10,
		)

		if err := first.Reschedule(&replacement); err != nil {
			t.Fatalf(
				"Reschedule() error = %v",
				err,
			)
		}

		var after Enrollment
		if err := db.First(
			&after,
			enrollment.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment: %v",
				err,
			)
		}

		if after.ClassesRemaining != before.ClassesRemaining {
			t.Fatalf(
				"ClassesRemaining after reschedule = %d, want %d",
				after.ClassesRemaining,
				before.ClassesRemaining,
			)
		}

		if after.Active != before.Active {
			t.Fatalf(
				"Active after reschedule = %t, want %t",
				after.Active,
				before.Active,
			)
		}

		var original Class
		if err := db.First(
			&original,
			first.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload original class: %v",
				err,
			)
		}

		if !original.Cancelled {
			t.Fatal(
				"original Cancelled = false, want true",
			)
		}

		if !original.CreditRefunded {
			t.Fatal(
				"original CreditRefunded = false, want true",
			)
		}

		var savedReplacement Class
		if err := db.First(
			&savedReplacement,
			replacement.ID,
		).Error; err != nil {
			t.Fatalf(
				"failed to reload replacement class: %v",
				err,
			)
		}

		if savedReplacement.Cancelled {
			t.Fatal(
				"replacement Cancelled = true, want false",
			)
		}

		if savedReplacement.CreditRefunded {
			t.Fatal(
				"replacement CreditRefunded = true, want false",
			)
		}
	})
}

func TestClassSaveIntegrity(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/class_integrity_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&Student{},
		&Course{},
		&Package{},
		&Enrollment{},
		&Class{},
	); err != nil {
		t.Fatalf("failed to migrate class test database: %v", err)
	}

	t.Cleanup(func() {
		uadmin.ClearDB()
		uadmin.Database = nil
	})

	student := Student{
		FirstName: "Test",
		LastName:  "Student",
		WeChatID:  "test-wechat",
		Email:     "test@example.com",
	}

	if _, err := student.Create(); err != nil {
		t.Fatalf("student.Create() error = %v", err)
	}

	course := Course{
		Title:  "Test Course",
		Active: true,
	}

	if err := db.Create(&course).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}

	now := time.Now()

	pkg := Package{
		Name:                   "Test Package",
		NumberOfClasses:        2,
		NumberOfFreeClasses:    0,
		TotalClasses:           2,
		ClassDurationInMinutes: 60,
		Price:                  100,
		ValidFrom:              &now,
		ValidUntil: func() *time.Time {
			value := now.Add(24 * time.Hour)
			return &value
		}(),
		Active: true,
	}

	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	enrollment := Enrollment{
		StudentID: student.ID,
		CourseID:  course.ID,
		PackageID: pkg.ID,
	}

	if err := enrollment.Create(); err != nil {
		t.Fatalf("enrollment.Create() error = %v", err)
	}

	classDate := now.Add(24 * time.Hour)

	startTime := time.Date(
		classDate.Year(),
		classDate.Month(),
		classDate.Day(),
		10,
		0,
		0,
		0,
		time.Local,
	)

	class := Class{
		ClassDate:    classDate,
		StartTime:    &startTime,
		EnrollmentID: enrollment.ID,
	}

	if err := class.Schedule(); err != nil {
		t.Fatalf("class.Schedule() error = %v", err)
	}

	originalEnrollmentID := class.EnrollmentID
	originalStudentID := class.StudentID
	originalCourseID := class.CourseID
	originalCancelled := class.Cancelled
	originalCreditRefunded := class.CreditRefunded
	originalPresent := class.Present
	originalAbsent := class.Absent
	originalStartTime := class.StartTime
	originalEndTime := class.EndTime
	originalRescheduledFromID := class.RescheduledFromID

	class.EnrollmentID = 999999
	class.StudentID = 999999
	class.CourseID = 999999
	class.Cancelled = true
	class.CreditRefunded = true
	class.Present = true
	class.Absent = true
	class.StartTime = nil
	class.EndTime = nil
	class.RescheduledFromID = 999999

	if err := db.Save(&class).Error; err != nil {
		t.Fatalf("save class: %v", err)
	}

	var saved Class

	if err := db.First(&saved, class.ID).Error; err != nil {
		t.Fatalf("reload class: %v", err)
	}

	if saved.EnrollmentID != originalEnrollmentID {
		t.Fatalf(
			"EnrollmentID = %d, want %d",
			saved.EnrollmentID,
			originalEnrollmentID,
		)
	}

	if saved.StudentID != originalStudentID {
		t.Fatalf(
			"StudentID = %d, want %d",
			saved.StudentID,
			originalStudentID,
		)
	}

	if saved.CourseID != originalCourseID {
		t.Fatalf(
			"CourseID = %d, want %d",
			saved.CourseID,
			originalCourseID,
		)
	}

	if saved.Cancelled != originalCancelled {
		t.Fatalf(
			"Cancelled = %v, want %v",
			saved.Cancelled,
			originalCancelled,
		)
	}

	if saved.CreditRefunded != originalCreditRefunded {
		t.Fatalf(
			"CreditRefunded = %v, want %v",
			saved.CreditRefunded,
			originalCreditRefunded,
		)
	}

	if saved.Present != originalPresent {
		t.Fatalf(
			"Present = %v, want %v",
			saved.Present,
			originalPresent,
		)
	}

	if saved.Absent != originalAbsent {
		t.Fatalf(
			"Absent = %v, want %v",
			saved.Absent,
			originalAbsent,
		)
	}

	if saved.StartTime == nil {
		t.Fatal("StartTime = nil, want protected value")
	}

	if originalStartTime == nil ||
		!saved.StartTime.Equal(*originalStartTime) {
		t.Fatal("StartTime changed")
	}

	if saved.EndTime == nil {
		t.Fatal("EndTime = nil, want protected value")
	}

	if originalEndTime == nil ||
		!saved.EndTime.Equal(*originalEndTime) {
		t.Fatal("EndTime changed")
	}

	if saved.RescheduledFromID != originalRescheduledFromID {
		t.Fatalf(
			"RescheduledFromID = %d, want %d",
			saved.RescheduledFromID,
			originalRescheduledFromID,
		)
	}
}

func TestClassSetAttendanceRejectsPresentWithRefund(t *testing.T) {
	setupClassScheduleTestDB(t)

	enrollment := createClassScheduleTestEnrollment(t, 5)

	class := newTestClass(enrollment.ID, 1, 10)

	if err := class.Schedule(); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	err := class.SetAttendance(true, true)
	if !errors.Is(err, ErrClassPresentCannotRefund) {
		t.Fatalf(
			"SetAttendance(true, true) error = %v, want %v",
			err,
			ErrClassPresentCannotRefund,
		)
	}

	var saved Class
	if err := uadmin.GetDB().First(&saved, class.ID).Error; err != nil {
		t.Fatalf("failed to reload class: %v", err)
	}

	if saved.Present {
		t.Fatal("Present = true, want false after rejected request")
	}

	if saved.CreditRefunded {
		t.Fatal("CreditRefunded = true, want false after rejected request")
	}
}

func TestClassSetAttendanceMarksAbsentWithoutRefund(t *testing.T) {
	db := setupClassScheduleTestDB(t)
	enrollment := createClassScheduleTestEnrollment(t, 5)

	class := newTestClass(enrollment.ID, 1, 10)

	if err := class.Schedule(); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	var before Enrollment
	if err := db.First(&before, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if before.ClassesRemaining != 4 {
		t.Fatalf(
			"ClassesRemaining before attendance = %d, want 4",
			before.ClassesRemaining,
		)
	}

	if err := class.SetAttendance(false, false); err != nil {
		t.Fatalf(
			"SetAttendance(false, false) error = %v",
			err,
		)
	}

	var savedClass Class
	if err := db.First(&savedClass, class.ID).Error; err != nil {
		t.Fatalf("failed to reload class: %v", err)
	}

	if savedClass.Present {
		t.Fatal("Present = true, want false")
	}

	if !savedClass.Absent {
		t.Fatal("Absent = false, want true")
	}

	if savedClass.CreditRefunded {
		t.Fatal("CreditRefunded = true, want false")
	}

	var after Enrollment
	if err := db.First(&after, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if after.ClassesRemaining != before.ClassesRemaining {
		t.Fatalf(
			"ClassesRemaining = %d, want %d",
			after.ClassesRemaining,
			before.ClassesRemaining,
		)
	}

	if after.Active != before.Active {
		t.Fatalf(
			"Active = %v, want %v",
			after.Active,
			before.Active,
		)
	}
}

func TestClassSetAttendanceAbsentWithRefundRestoresCredit(t *testing.T) {
	db := setupClassScheduleTestDB(t)
	enrollment := createClassScheduleTestEnrollment(t, 5)

	class := newTestClass(enrollment.ID, 1, 10)

	if err := class.Schedule(); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	var before Enrollment
	if err := db.First(&before, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if before.ClassesRemaining != 4 {
		t.Fatalf(
			"ClassesRemaining before attendance = %d, want 4",
			before.ClassesRemaining,
		)
	}

	if err := class.SetAttendance(false, true); err != nil {
		t.Fatalf(
			"SetAttendance(false, true) error = %v",
			err,
		)
	}

	var savedClass Class
	if err := db.First(&savedClass, class.ID).Error; err != nil {
		t.Fatalf("failed to reload class: %v", err)
	}

	if savedClass.Present {
		t.Fatal("Present = true, want false")
	}

	if !savedClass.Absent {
		t.Fatal("Absent = false, want true")
	}

	if !savedClass.CreditRefunded {
		t.Fatal("CreditRefunded = false, want true")
	}

	var after Enrollment
	if err := db.First(&after, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if after.ClassesRemaining != 5 {
		t.Fatalf(
			"ClassesRemaining = %d, want 5",
			after.ClassesRemaining,
		)
	}

	if !after.Active {
		t.Fatal("Active = false, want true after credit refund")
	}
}

func TestClassSetAttendanceRejectsDuplicateRecording(t *testing.T) {
	db := setupClassScheduleTestDB(t)
	enrollment := createClassScheduleTestEnrollment(t, 5)

	class := newTestClass(enrollment.ID, 1, 10)

	if err := class.Schedule(); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	if err := class.SetAttendance(false, true); err != nil {
		t.Fatalf(
			"first SetAttendance() error = %v",
			err,
		)
	}

	var before Enrollment
	if err := db.First(&before, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if before.ClassesRemaining != 5 {
		t.Fatalf(
			"ClassesRemaining after first attendance = %d, want 5",
			before.ClassesRemaining,
		)
	}

	err := class.SetAttendance(false, false)

	if !errors.Is(err, ErrClassAttendanceRecorded) {
		t.Fatalf(
			"second SetAttendance() error = %v, want %v",
			err,
			ErrClassAttendanceRecorded,
		)
	}

	var after Enrollment
	if err := db.First(&after, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if after.ClassesRemaining != before.ClassesRemaining {
		t.Fatalf(
			"ClassesRemaining after rejected duplicate = %d, want %d",
			after.ClassesRemaining,
			before.ClassesRemaining,
		)
	}

	var saved Class
	if err := db.First(&saved, class.ID).Error; err != nil {
		t.Fatalf("failed to reload class: %v", err)
	}

	if !saved.Absent {
		t.Fatal("Absent = false, want true")
	}

	if !saved.CreditRefunded {
		t.Fatal("CreditRefunded = false, want true")
	}
}

func TestClassCancelDoesNotRefundCredit(t *testing.T) {
	db := setupClassScheduleTestDB(t)
	enrollment := createClassScheduleTestEnrollment(t, 5)

	class := newTestClass(enrollment.ID, 1, 10)

	if err := class.Schedule(); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	var before Enrollment
	if err := db.First(&before, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if before.ClassesRemaining != 4 {
		t.Fatalf(
			"ClassesRemaining before cancel = %d, want 4",
			before.ClassesRemaining,
		)
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

	var after Enrollment
	if err := db.First(&after, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if after.ClassesRemaining != before.ClassesRemaining {
		t.Fatalf(
			"ClassesRemaining after cancel = %d, want %d",
			after.ClassesRemaining,
			before.ClassesRemaining,
		)
	}

	if after.Active != before.Active {
		t.Fatalf(
			"Active after cancel = %v, want %v",
			after.Active,
			before.Active,
		)
	}
}

func TestClassRefundCreditRestoresCreditAfterCancel(t *testing.T) {
	db := setupClassScheduleTestDB(t)
	enrollment := createClassScheduleTestEnrollment(t, 5)

	class := newTestClass(enrollment.ID, 1, 10)

	if err := class.Schedule(); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	if err := class.Cancel(); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	var before Enrollment
	if err := db.First(&before, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if before.ClassesRemaining != 4 {
		t.Fatalf(
			"ClassesRemaining before refund = %d, want 4",
			before.ClassesRemaining,
		)
	}

	if err := class.RefundCredit(); err != nil {
		t.Fatalf("RefundCredit() error = %v", err)
	}

	var savedClass Class
	if err := db.First(&savedClass, class.ID).Error; err != nil {
		t.Fatalf("failed to reload class: %v", err)
	}

	if !savedClass.Cancelled {
		t.Fatal("Cancelled = false, want true")
	}

	if !savedClass.CreditRefunded {
		t.Fatal("CreditRefunded = false, want true")
	}

	var after Enrollment
	if err := db.First(&after, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if after.ClassesRemaining != 5 {
		t.Fatalf(
			"ClassesRemaining after refund = %d, want 5",
			after.ClassesRemaining,
		)
	}

	if !after.Active {
		t.Fatal("Active = false, want true")
	}
}

func TestClassRefundCreditRejectsDuplicateRefund(t *testing.T) {
	db := setupClassScheduleTestDB(t)
	enrollment := createClassScheduleTestEnrollment(t, 5)

	class := newTestClass(enrollment.ID, 1, 10)

	if err := class.Schedule(); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	if err := class.Cancel(); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	if err := class.RefundCredit(); err != nil {
		t.Fatalf("first RefundCredit() error = %v", err)
	}

	var before Enrollment
	if err := db.First(&before, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if before.ClassesRemaining != 5 {
		t.Fatalf(
			"ClassesRemaining before duplicate refund = %d, want 5",
			before.ClassesRemaining,
		)
	}

	err := class.RefundCredit()
	if !errors.Is(err, ErrClassCreditAlreadyRefunded) {
		t.Fatalf(
			"second RefundCredit() error = %v, want %v",
			err,
			ErrClassCreditAlreadyRefunded,
		)
	}

	var after Enrollment
	if err := db.First(&after, enrollment.ID).Error; err != nil {
		t.Fatalf("failed to reload enrollment: %v", err)
	}

	if after.ClassesRemaining != before.ClassesRemaining {
		t.Fatalf(
			"ClassesRemaining after duplicate refund = %d, want %d",
			after.ClassesRemaining,
			before.ClassesRemaining,
		)
	}
}
