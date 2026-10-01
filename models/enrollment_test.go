package models

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

func setupEnrollmentCreateTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/enrollment_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&uadmin.User{},
		&Level{},
		&Student{},
		&Course{},
		&Package{},
		&Enrollment{},
	); err != nil {
		t.Fatalf(
			"failed to migrate enrollment test database: %v",
			err,
		)
	}

	t.Cleanup(func() {
		uadmin.ClearDB()
		uadmin.Database = nil
	})

	return db
}

func createEnrollmentTestStudent(t *testing.T) Student {
	t.Helper()

	student := Student{
		FirstName: "John",
		LastName:  "Doe",
		WeChatID:  "wechat-test",
		Email:     "john@example.com",
	}

	if err := uadmin.Save(&student); err != nil {
		t.Fatalf("failed to create test student: %v", err)
	}

	return student
}

func createEnrollmentTestCourse(t *testing.T) Course {
	t.Helper()

	course := Course{
		Title:       "General English",
		Description: "General English course",
		Active:      true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("failed to create test course: %v", err)
	}

	return course
}

func createEnrollmentTestPackage(
	t *testing.T,
	numberOfClasses int,
	numberOfFreeClasses int,
) Package {
	t.Helper()

	validFrom := time.Now()
	validUntil := validFrom.AddDate(0, 6, 0)

	pkg := Package{
		Name:                   "Test Package",
		NumberOfClasses:        numberOfClasses,
		NumberOfFreeClasses:    numberOfFreeClasses,
		ClassDurationInMinutes: 60,
		Price:                  1000,
		ValidFrom:              &validFrom,
		ValidUntil:             &validUntil,
		Active:                 true,
	}

	pkg.Save()

	if pkg.ID == 0 {
		t.Fatal("expected test package to be saved")
	}

	return pkg
}

func TestEnrollmentCreate(t *testing.T) {
	t.Run("creates enrollment with generated reference and derived class counts", func(t *testing.T) {
		db := setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 8, 2)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := enrollment.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if enrollment.ID == 0 {
			t.Fatal("expected enrollment ID to be assigned")
		}

		if enrollment.ReferenceNumber == "" {
			t.Fatal("expected reference number to be generated")
		}

		if matched, err := regexp.MatchString(
			`^[A-Z0-9]{12}$`,
			enrollment.ReferenceNumber,
		); err != nil {
			t.Fatalf("reference regex error: %v", err)
		} else if !matched {
			t.Fatalf(
				"reference number = %q, want 12 uppercase alphanumeric characters",
				enrollment.ReferenceNumber,
			)
		}

		if enrollment.TotalClasses != 10 {
			t.Fatalf(
				"TotalClasses = %d, want 10",
				enrollment.TotalClasses,
			)
		}

		if enrollment.ClassesRemaining != 10 {
			t.Fatalf(
				"ClassesRemaining = %d, want 10",
				enrollment.ClassesRemaining,
			)
		}

		var saved Enrollment

		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if saved.ReferenceNumber != enrollment.ReferenceNumber {
			t.Fatalf(
				"saved ReferenceNumber = %q, want %q",
				saved.ReferenceNumber,
				enrollment.ReferenceNumber,
			)
		}

		if saved.TotalClasses != 10 {
			t.Fatalf(
				"saved TotalClasses = %d, want 10",
				saved.TotalClasses,
			)
		}

		if saved.ClassesRemaining != 10 {
			t.Fatalf(
				"saved ClassesRemaining = %d, want 10",
				saved.ClassesRemaining,
			)
		}
	})

	t.Run("generates unique references", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		first := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := first.Create(); err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		second := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := second.Create(); err != nil {
			t.Fatalf("second Create() error = %v", err)
		}

		if first.ReferenceNumber == second.ReferenceNumber {
			t.Fatalf(
				"generated duplicate reference number %q",
				first.ReferenceNumber,
			)
		}
	})

	t.Run("preserves supplied reference number", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 2)

		const reference = "CUSTOMREF12"

		enrollment := Enrollment{
			ReferenceNumber: reference,
			StudentID:       student.ID,
			CourseID:        course.ID,
			PackageID:       pkg.ID,
		}

		if err := enrollment.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if enrollment.ReferenceNumber != reference {
			t.Fatalf(
				"ReferenceNumber = %q, want %q",
				enrollment.ReferenceNumber,
				reference,
			)
		}
	})

	t.Run("rejects duplicate supplied reference", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		const reference = "DUPLICATEREF"

		first := Enrollment{
			ReferenceNumber: reference,
			StudentID:       student.ID,
			CourseID:        course.ID,
			PackageID:       pkg.ID,
		}

		if err := first.Create(); err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		second := Enrollment{
			ReferenceNumber: reference,
			StudentID:       student.ID,
			CourseID:        course.ID,
			PackageID:       pkg.ID,
		}

		err := second.Create()

		if !errors.Is(err, ErrEnrollmentReferenceExists) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentReferenceExists,
			)
		}
	})

	t.Run("rejects missing student", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		enrollment := Enrollment{
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentStudentRequired) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentStudentRequired,
			)
		}
	})

	t.Run("rejects missing course", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		enrollment := Enrollment{
			StudentID: student.ID,
			PackageID: pkg.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentCourseRequired) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentCourseRequired,
			)
		}
	})

	t.Run("rejects missing package", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentPackageRequired) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentPackageRequired,
			)
		}
	})

	t.Run("rejects nonexistent student", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		enrollment := Enrollment{
			StudentID: 99999,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentStudentNotFound) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentStudentNotFound,
			)
		}
	})

	t.Run("rejects nonexistent course", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  99999,
			PackageID: pkg.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentCourseNotFound) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentCourseNotFound,
			)
		}
	})

	t.Run("rejects nonexistent package", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: 99999,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentPackageNotFound) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentPackageNotFound,
			)
		}
	})

	t.Run("rejects package with no classes", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 0, 0)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentPackageNoClasses) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentPackageNoClasses,
			)
		}
	})

	t.Run("does not persist invalid enrollment", func(t *testing.T) {
		db := setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 0, 0)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := enrollment.Create(); err == nil {
			t.Fatal("Create() error = nil, want package validation error")
		}

		var count int64

		if err := db.Model(&Enrollment{}).Count(&count).Error; err != nil {
			t.Fatalf("failed to count enrollments: %v", err)
		}

		if count != 0 {
			t.Fatalf(
				"enrollment count = %d, want 0",
				count,
			)
		}
	})

	t.Run("does not overwrite package-derived class counts", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 8, 2)

		enrollment := Enrollment{
			StudentID:        student.ID,
			CourseID:         course.ID,
			PackageID:        pkg.ID,
			TotalClasses:     999,
			ClassesRemaining: 1,
		}

		if err := enrollment.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if enrollment.TotalClasses != pkg.TotalClasses {
			t.Fatalf(
				"TotalClasses = %d, want package TotalClasses %d",
				enrollment.TotalClasses,
				pkg.TotalClasses,
			)
		}

		if enrollment.ClassesRemaining != pkg.TotalClasses {
			t.Fatalf(
				"ClassesRemaining = %d, want package TotalClasses %d",
				enrollment.ClassesRemaining,
				pkg.TotalClasses,
			)
		}
	})
}
