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

func TestClassRefundCredit(t *testing.T) {
	t.Run("refunds one credit from a cancelled class", func(t *testing.T) {
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

		var savedEnrollment Enrollment

		if err := db.First(&savedEnrollment, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
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

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf("Cancel() error = %v", err)
		}

		var exhausted Enrollment

		if err := db.First(&exhausted, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
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
			t.Fatalf("RefundCredit() error = %v", err)
		}

		var saved Enrollment

		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
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

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
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

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf("Cancel() error = %v", err)
		}

		if err := class.RefundCredit(); err != nil {
			t.Fatalf("first RefundCredit() error = %v", err)
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

	t.Run("preserves enrollment course when refunding credit", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 5)
		originalCourseID := enrollment.CourseID

		class := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		if err := class.Cancel(); err != nil {
			t.Fatalf("Cancel() error = %v", err)
		}

		if err := class.RefundCredit(); err != nil {
			t.Fatalf("RefundCredit() error = %v", err)
		}

		var saved Enrollment

		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if saved.CourseID != originalCourseID {
			t.Fatalf(
				"CourseID = %d, want %d",
				saved.CourseID,
				originalCourseID,
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
		db := setupClassScheduleTestDB(t)

		class := Class{
			EnrollmentID: 99999,
			Cancelled:    true,
		}

		if err := db.Create(&class).Error; err != nil {
			t.Fatalf("failed to create test class: %v", err)
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
}

func TestClassReschedule(t *testing.T) {
	t.Run("cancels original class and creates replacement", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 5)

		original := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := original.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		var originalBefore Class

		if err := db.First(&originalBefore, original.ID).Error; err != nil {
			t.Fatalf("failed to reload original class: %v", err)
		}

		replacementDate := time.Now().AddDate(0, 0, 1)
		replacementStart := time.Now().Add(24 * time.Hour)
		replacementEnd := replacementStart.Add(time.Hour)

		replacement := Class{
			ClassDate: replacementDate,
			StartTime: &replacementStart,
			EndTime:   &replacementEnd,
		}

		if err := original.Reschedule(&replacement); err != nil {
			t.Fatalf("Reschedule() error = %v", err)
		}

		var savedOriginal Class

		if err := db.First(&savedOriginal, original.ID).Error; err != nil {
			t.Fatalf("failed to reload original class: %v", err)
		}

		if !savedOriginal.Cancelled {
			t.Fatal("original Cancelled = false, want true")
		}

		if !savedOriginal.CreditRefunded {
			t.Fatal("original CreditRefunded = false, want true")
		}

		var savedReplacement Class

		if err := db.First(&savedReplacement, replacement.ID).Error; err != nil {
			t.Fatalf("failed to reload replacement class: %v", err)
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

		original := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := original.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		var before Enrollment

		if err := db.First(&before, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		replacement := Class{
			ClassDate: time.Now().AddDate(0, 0, 1),
		}

		if err := original.Reschedule(&replacement); err != nil {
			t.Fatalf("Reschedule() error = %v", err)
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
				"Active = %t, want %t",
				after.Active,
				before.Active,
			)
		}
	})

	t.Run("rolls back when replacement creation fails", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 5)

		original := Class{
			EnrollmentID: enrollment.ID,
		}

		if err := original.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		var beforeEnrollment Enrollment

		if err := db.First(&beforeEnrollment, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		var beforeClass Class

		if err := db.First(&beforeClass, original.ID).Error; err != nil {
			t.Fatalf("failed to reload original class: %v", err)
		}

		/*
			Using the original class ID forces the replacement INSERT
			to fail because the primary key already exists.
		*/
		replacement := Class{
			Model: uadmin.Model{
				ID: original.ID,
			},
			ClassDate: time.Now().AddDate(0, 0, 1),
		}

		err := original.Reschedule(&replacement)

		if !errors.Is(err, ErrClassRescheduleFailed) {
			t.Fatalf(
				"Reschedule() error = %v, want wrapped ErrClassRescheduleFailed",
				err,
			)
		}

		/*
			The entire transaction must have rolled back.
		*/

		var afterClass Class

		if err := db.First(&afterClass, original.ID).Error; err != nil {
			t.Fatalf("failed to reload original class after rollback: %v", err)
		}

		if afterClass.Cancelled != beforeClass.Cancelled {
			t.Fatalf(
				"Cancelled after rollback = %t, want %t",
				afterClass.Cancelled,
				beforeClass.Cancelled,
			)
		}

		if afterClass.CreditRefunded != beforeClass.CreditRefunded {
			t.Fatalf(
				"CreditRefunded after rollback = %t, want %t",
				afterClass.CreditRefunded,
				beforeClass.CreditRefunded,
			)
		}

		var afterEnrollment Enrollment

		if err := db.First(&afterEnrollment, enrollment.ID).Error; err != nil {
			t.Fatalf(
				"failed to reload enrollment after rollback: %v",
				err,
			)
		}

		if afterEnrollment.ClassesRemaining != beforeEnrollment.ClassesRemaining {
			t.Fatalf(
				"ClassesRemaining after rollback = %d, want %d",
				afterEnrollment.ClassesRemaining,
				beforeEnrollment.ClassesRemaining,
			)
		}

		if afterEnrollment.Active != beforeEnrollment.Active {
			t.Fatalf(
				"Active after rollback = %t, want %t",
				afterEnrollment.Active,
				beforeEnrollment.Active,
			)
		}

		/*
			No replacement class should have been created.
			The original class ID is still the only class involved.
		*/
		var classCount int64

		if err := db.Model(&Class{}).
			Where("enrollment_id = ?", enrollment.ID).
			Count(&classCount).Error; err != nil {
			t.Fatalf("failed to count enrollment classes: %v", err)
		}

		if classCount != 1 {
			t.Fatalf(
				"class count after rollback = %d, want 1",
				classCount,
			)
		}
	})

}
