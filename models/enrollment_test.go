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
		cleanupTestDB(t, db)
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
		firstCourse := createEnrollmentTestCourse(t)

		secondCourse := Course{
			Title:  "Business English",
			Active: true,
		}

		if err := uadmin.Save(&secondCourse); err != nil {
			t.Fatalf("failed to create second test course: %v", err)
		}

		pkg := createEnrollmentTestPackage(t, 10, 0)

		first := Enrollment{
			StudentID: student.ID,
			CourseID:  firstCourse.ID,
			PackageID: pkg.ID,
		}

		if err := first.Create(); err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		second := Enrollment{
			StudentID: student.ID,
			CourseID:  secondCourse.ID,
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
		firstCourse := createEnrollmentTestCourse(t)

		secondCourse := Course{
			Title:  "Business English",
			Active: true,
		}

		if err := uadmin.Save(&secondCourse); err != nil {
			t.Fatalf("failed to create second test course: %v", err)
		}

		pkg := createEnrollmentTestPackage(t, 10, 0)

		const reference = "DUPLICATEREF"

		first := Enrollment{
			ReferenceNumber: reference,
			StudentID:       student.ID,
			CourseID:        firstCourse.ID,
			PackageID:       pkg.ID,
		}

		if err := first.Create(); err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		second := Enrollment{
			ReferenceNumber: reference,
			StudentID:       student.ID,
			CourseID:        secondCourse.ID,
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

	t.Run("creates active enrollment", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := enrollment.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if !enrollment.Active {
			t.Fatal("Active = false, want true")
		}
	})

	t.Run("rejects inactive course", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		course.Active = false

		if err := uadmin.Save(&course); err != nil {
			t.Fatalf("failed to deactivate test course: %v", err)
		}

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentCourseInactive) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentCourseInactive,
			)
		}
	})

	t.Run("rejects inactive package", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		db := uadmin.GetDB()

		pkg = updateEnrollmentPackage(t, db, pkg, map[string]any{
			"active": false,
		})

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentPackageInactive) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentPackageInactive,
			)
		}
	})

	t.Run("rejects package not yet valid", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		db := uadmin.GetDB()

		validFrom := time.Now().Add(24 * time.Hour)

		pkg = updateEnrollmentPackage(t, db, pkg, map[string]any{
			"valid_from": validFrom,
		})

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentPackageExpired) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentPackageExpired,
			)
		}
	})

	t.Run("rejects expired package", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		db := uadmin.GetDB()

		validUntil := time.Now().Add(-24 * time.Hour)

		pkg = updateEnrollmentPackage(t, db, pkg, map[string]any{
			"valid_until": validUntil,
		})

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		err := enrollment.Create()

		if !errors.Is(err, ErrEnrollmentPackageExpired) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentPackageExpired,
			)
		}
	})

	t.Run("allows new enrollment when previous enrollment is inactive", func(t *testing.T) {
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

		db := uadmin.GetDB()

		if err := db.Model(&first).Update("active", false).Error; err != nil {
			t.Fatalf("failed to deactivate first enrollment: %v", err)
		}

		second := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := second.Create(); err != nil {
			t.Fatalf("second Create() error = %v", err)
		}

		if !second.Active {
			t.Fatal("second enrollment Active = false, want true")
		}
	})

	t.Run("rejects duplicate active enrollment for same student and course", func(t *testing.T) {
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

		err := second.Create()

		if !errors.Is(err, ErrEnrollmentAlreadyExists) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrEnrollmentAlreadyExists,
			)
		}
	})

	t.Run("allows same student to enroll in a different course", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		firstCourse := createEnrollmentTestCourse(t)
		secondCourse := Course{
			Title:  "Business English",
			Active: true,
		}

		if err := uadmin.Save(&secondCourse); err != nil {
			t.Fatalf("failed to create second test course: %v", err)
		}

		pkg := createEnrollmentTestPackage(t, 10, 0)

		first := Enrollment{
			StudentID: student.ID,
			CourseID:  firstCourse.ID,
			PackageID: pkg.ID,
		}

		if err := first.Create(); err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		second := Enrollment{
			StudentID: student.ID,
			CourseID:  secondCourse.ID,
			PackageID: pkg.ID,
		}

		if err := second.Create(); err != nil {
			t.Fatalf("second Create() error = %v", err)
		}
	})

	t.Run("existing enrollment remains valid when package becomes inactive", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := enrollment.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		db := uadmin.GetDB()

		if err := db.Model(&pkg).Update("active", false).Error; err != nil {
			t.Fatalf("failed to deactivate package: %v", err)
		}

		var saved Enrollment

		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if !saved.Active {
			t.Fatal("existing enrollment became inactive")
		}

		if saved.PackageID != pkg.ID {
			t.Fatalf(
				"PackageID = %d, want %d",
				saved.PackageID,
				pkg.ID,
			)
		}

		if saved.ClassesRemaining != 10 {
			t.Fatalf(
				"ClassesRemaining = %d, want 10",
				saved.ClassesRemaining,
			)
		}
	})

	t.Run("changes course successfully", func(t *testing.T) {
		db := setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		firstCourse := createEnrollmentTestCourse(t)

		secondCourse := Course{
			Title:  "Business English",
			Active: true,
		}

		if err := uadmin.Save(&secondCourse); err != nil {
			t.Fatalf("failed to create second test course: %v", err)
		}

		pkg := createEnrollmentTestPackage(t, 10, 0)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  firstCourse.ID,
			PackageID: pkg.ID,
		}

		if err := enrollment.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		originalRemaining := enrollment.ClassesRemaining
		originalTotal := enrollment.TotalClasses
		originalPackage := enrollment.PackageID

		if err := enrollment.ChangeCourse(secondCourse.ID); err != nil {
			t.Fatalf("ChangeCourse() error = %v", err)
		}

		if enrollment.CourseID != secondCourse.ID {
			t.Fatalf(
				"CourseID = %d, want %d",
				enrollment.CourseID,
				secondCourse.ID,
			)
		}

		if enrollment.ClassesRemaining != originalRemaining {
			t.Fatalf(
				"ClassesRemaining = %d, want %d",
				enrollment.ClassesRemaining,
				originalRemaining,
			)
		}

		var saved Enrollment

		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if saved.CourseID != secondCourse.ID {
			t.Fatalf(
				"CourseID = %d, want %d",
				saved.CourseID,
				secondCourse.ID,
			)
		}

		if saved.ClassesRemaining != originalRemaining {
			t.Fatalf(
				"ClassesRemaining = %d, want %d",
				saved.ClassesRemaining,
				originalRemaining,
			)
		}

		if saved.TotalClasses != originalTotal {
			t.Fatalf(
				"TotalClasses = %d, want %d",
				saved.TotalClasses,
				originalTotal,
			)
		}

		if saved.PackageID != originalPackage {
			t.Fatalf(
				"PackageID = %d, want %d",
				saved.PackageID,
				originalPackage,
			)
		}
	})

	t.Run("preserves historical course when enrollment course changes", func(t *testing.T) {
		db := setupClassScheduleTestDB(t)

		enrollment := createClassScheduleTestEnrollment(t, 5)

		originalCourseID := enrollment.CourseID

		secondCourse := Course{
			Title:  "Business English",
			Active: true,
		}

		if err := uadmin.Save(&secondCourse); err != nil {
			t.Fatalf("failed to create second test course: %v", err)
		}

		class := newTestClass(enrollment.ID, 5, 14)
		if err := class.Schedule(); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}

		if err := enrollment.ChangeCourse(secondCourse.ID); err != nil {
			t.Fatalf("ChangeCourse() error = %v", err)
		}

		var savedEnrollment Enrollment

		if err := db.First(&savedEnrollment, enrollment.ID).Error; err != nil {
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

		var savedClass Class

		if err := db.First(&savedClass, class.ID).Error; err != nil {
			t.Fatalf(
				"failed to reload class: %v",
				err,
			)
		}

		if savedClass.CourseID != originalCourseID {
			t.Fatalf(
				"class CourseID = %d, want historical course %d",
				savedClass.CourseID,
				originalCourseID,
			)
		}
	})
}

func updateEnrollmentPackage(
	t *testing.T,
	db *gorm.DB,
	pkg Package,
	updates map[string]any,
) Package {
	t.Helper()

	if err := db.Model(&pkg).Updates(updates).Error; err != nil {
		t.Fatalf("failed to update test package: %v", err)
	}

	if err := db.First(&pkg, pkg.ID).Error; err != nil {
		t.Fatalf("failed to reload test package: %v", err)
	}

	return pkg
}

func TestEnrollmentSaveIntegrity(t *testing.T) {
	t.Run("generic save can currently change protected enrollment fields", func(t *testing.T) {
		db := setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)

		otherStudent := Student{
			FirstName: "Jane",
			LastName:  "Smith",
			WeChatID:  "wechat-other",
			Email:     "jane@example.com",
		}
		if err := uadmin.Save(&otherStudent); err != nil {
			t.Fatalf("failed to create second test student: %v", err)
		}

		course := createEnrollmentTestCourse(t)

		otherCourse := Course{
			Title:  "Business English",
			Active: true,
		}
		if err := uadmin.Save(&otherCourse); err != nil {
			t.Fatalf("failed to create second test course: %v", err)
		}

		pkg := createEnrollmentTestPackage(t, 10, 0)
		otherPackage := createEnrollmentTestPackage(t, 20, 0)

		enrollment := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
			Contract:  "original-contract",
		}

		if err := enrollment.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		originalReference := enrollment.ReferenceNumber
		originalTotal := enrollment.TotalClasses
		originalRemaining := enrollment.ClassesRemaining

		// Attempt to bypass the domain operations by directly modifying
		// fields and using the generic persistence path.
		enrollment.StudentID = otherStudent.ID
		enrollment.CourseID = otherCourse.ID
		enrollment.PackageID = otherPackage.ID
		enrollment.TotalClasses = 999
		enrollment.ClassesRemaining = 999
		enrollment.Active = false
		enrollment.ReferenceNumber = "CHANGEDREF12"
		enrollment.Contract = "updated-contract"

		if err := uadmin.Save(&enrollment); err != nil {
			t.Fatalf("uadmin.Save() error = %v", err)
		}

		var saved Enrollment
		if err := db.First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		// These assertions intentionally describe the integrity rules
		// we want to enforce. They should currently fail until the
		// Enrollment persistence protection is implemented.
		if saved.StudentID != student.ID {
			t.Fatalf(
				"StudentID = %d, want protected value %d",
				saved.StudentID,
				student.ID,
			)
		}

		if saved.CourseID != course.ID {
			t.Fatalf(
				"CourseID = %d, want protected value %d",
				saved.CourseID,
				course.ID,
			)
		}

		if saved.PackageID != pkg.ID {
			t.Fatalf(
				"PackageID = %d, want protected value %d",
				saved.PackageID,
				pkg.ID,
			)
		}

		if saved.TotalClasses != originalTotal {
			t.Fatalf(
				"TotalClasses = %d, want protected value %d",
				saved.TotalClasses,
				originalTotal,
			)
		}

		if saved.ClassesRemaining != originalRemaining {
			t.Fatalf(
				"ClassesRemaining = %d, want protected value %d",
				saved.ClassesRemaining,
				originalRemaining,
			)
		}

		if !saved.Active {
			t.Fatalf(
				"Active = %v, want protected value true",
				saved.Active,
			)
		}

		if saved.ReferenceNumber != originalReference {
			t.Fatalf(
				"ReferenceNumber = %q, want protected value %q",
				saved.ReferenceNumber,
				originalReference,
			)
		}

		if saved.Contract != "updated-contract" {
			t.Fatalf(
				"Contract = %q, want updated value %q",
				saved.Contract,
				"updated-contract",
			)
		}
	})
}

func TestEnrollmentDeactivate(t *testing.T) {
	t.Run("deactivates active enrollment and preserves enrollment data", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

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

		reference := enrollment.ReferenceNumber
		totalClasses := enrollment.TotalClasses
		classesRemaining := enrollment.ClassesRemaining
		studentID := enrollment.StudentID
		courseID := enrollment.CourseID
		packageID := enrollment.PackageID

		if err := enrollment.Deactivate(); err != nil {
			t.Fatalf("Deactivate() error = %v", err)
		}

		if enrollment.Active {
			t.Fatal("Active = true, want false")
		}

		if enrollment.ReferenceNumber != reference {
			t.Fatalf(
				"ReferenceNumber = %q, want %q",
				enrollment.ReferenceNumber,
				reference,
			)
		}

		if enrollment.TotalClasses != totalClasses {
			t.Fatalf(
				"TotalClasses = %d, want %d",
				enrollment.TotalClasses,
				totalClasses,
			)
		}

		if enrollment.ClassesRemaining != classesRemaining {
			t.Fatalf(
				"ClassesRemaining = %d, want %d",
				enrollment.ClassesRemaining,
				classesRemaining,
			)
		}

		if enrollment.StudentID != studentID {
			t.Fatalf(
				"StudentID = %d, want %d",
				enrollment.StudentID,
				studentID,
			)
		}

		if enrollment.CourseID != courseID {
			t.Fatalf(
				"CourseID = %d, want %d",
				enrollment.CourseID,
				courseID,
			)
		}

		if enrollment.PackageID != packageID {
			t.Fatalf(
				"PackageID = %d, want %d",
				enrollment.PackageID,
				packageID,
			)
		}

		var saved Enrollment

		if err := uadmin.GetDB().First(&saved, enrollment.ID).Error; err != nil {
			t.Fatalf("failed to reload enrollment: %v", err)
		}

		if saved.Active {
			t.Fatal("saved Active = true, want false")
		}

		if saved.ClassesRemaining != classesRemaining {
			t.Fatalf(
				"saved ClassesRemaining = %d, want %d",
				saved.ClassesRemaining,
				classesRemaining,
			)
		}
	})

	t.Run("rejects nonexistent enrollment", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		enrollment := Enrollment{
			Model: uadmin.Model{
				ID: 99999,
			},
		}

		err := enrollment.Deactivate()

		if !errors.Is(err, ErrEnrollmentNotFound) {
			t.Fatalf(
				"Deactivate() error = %v, want %v",
				err,
				ErrEnrollmentNotFound,
			)
		}
	})

	t.Run("rejects already inactive enrollment", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

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

		if err := uadmin.GetDB().
			Model(&Enrollment{}).
			Where("id = ?", enrollment.ID).
			Update("active", false).Error; err != nil {
			t.Fatalf("failed to deactivate test enrollment: %v", err)
		}

		err := enrollment.Deactivate()

		if !errors.Is(err, ErrEnrollmentDeactivateAlreadyInactive) {
			t.Fatalf(
				"Deactivate() error = %v, want %v",
				err,
				ErrEnrollmentDeactivateAlreadyInactive,
			)
		}
	})
}

func TestEnrollmentRenewWithTx(t *testing.T) {
	t.Run("creates a new enrollment and preserves the original", func(t *testing.T) {
		db := setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		originalCourse := createEnrollmentTestCourse(t)

		newCourse := Course{
			Title:  "Business English",
			Active: true,
		}

		if err := uadmin.Save(&newCourse); err != nil {
			t.Fatalf("failed to create renewal course: %v", err)
		}

		originalPackage := createEnrollmentTestPackage(t, 8, 2)
		renewalPackage := createEnrollmentTestPackage(t, 15, 3)

		original := Enrollment{
			StudentID: student.ID,
			CourseID:  originalCourse.ID,
			PackageID: originalPackage.ID,
			Contract:  "original-contract.pdf",
		}

		if err := original.Create(); err != nil {
			t.Fatalf("original Create() error = %v", err)
		}

		originalReference := original.ReferenceNumber
		originalTotal := original.TotalClasses
		originalRemaining := original.ClassesRemaining
		originalCourseID := original.CourseID
		originalPackageID := original.PackageID
		originalContract := original.Contract

		var renewed *Enrollment

		err := db.Transaction(func(tx *gorm.DB) error {
			var err error

			renewed, err = original.RenewWithTx(
				tx,
				newCourse.ID,
				renewalPackage.ID,
				"renewed-contract.pdf",
			)

			return err
		})

		if err != nil {
			t.Fatalf("RenewWithTx() error = %v", err)
		}

		if renewed == nil {
			t.Fatal("expected renewed enrollment")
		}

		if renewed.ID == 0 {
			t.Fatal("expected renewed enrollment ID")
		}

		if renewed.ID == original.ID {
			t.Fatal("renewed enrollment reused original ID")
		}

		if renewed.ReferenceNumber == "" {
			t.Fatal("expected renewed reference number")
		}

		if renewed.ReferenceNumber == originalReference {
			t.Fatalf(
				"renewed reference = %q, same as original",
				renewed.ReferenceNumber,
			)
		}

		if renewed.StudentID != student.ID {
			t.Fatalf(
				"renewed StudentID = %d, want %d",
				renewed.StudentID,
				student.ID,
			)
		}

		if renewed.CourseID != newCourse.ID {
			t.Fatalf(
				"renewed CourseID = %d, want %d",
				renewed.CourseID,
				newCourse.ID,
			)
		}

		if renewed.PackageID != renewalPackage.ID {
			t.Fatalf(
				"renewed PackageID = %d, want %d",
				renewed.PackageID,
				renewalPackage.ID,
			)
		}

		if renewed.TotalClasses != renewalPackage.TotalClasses {
			t.Fatalf(
				"renewed TotalClasses = %d, want %d",
				renewed.TotalClasses,
				renewalPackage.TotalClasses,
			)
		}

		if renewed.ClassesRemaining != renewalPackage.TotalClasses {
			t.Fatalf(
				"renewed ClassesRemaining = %d, want %d",
				renewed.ClassesRemaining,
				renewalPackage.TotalClasses,
			)
		}

		if !renewed.Active {
			t.Fatal("renewed enrollment Active = false, want true")
		}

		if renewed.Contract != "renewed-contract.pdf" {
			t.Fatalf(
				"renewed Contract = %q, want %q",
				renewed.Contract,
				"renewed-contract.pdf",
			)
		}

		var savedOriginal Enrollment

		if err := db.First(&savedOriginal, original.ID).Error; err != nil {
			t.Fatalf(
				"failed to reload original enrollment: %v",
				err,
			)
		}

		if savedOriginal.ReferenceNumber != originalReference {
			t.Fatalf(
				"original ReferenceNumber = %q, want %q",
				savedOriginal.ReferenceNumber,
				originalReference,
			)
		}

		if savedOriginal.CourseID != originalCourseID {
			t.Fatalf(
				"original CourseID = %d, want %d",
				savedOriginal.CourseID,
				originalCourseID,
			)
		}

		if savedOriginal.PackageID != originalPackageID {
			t.Fatalf(
				"original PackageID = %d, want %d",
				savedOriginal.PackageID,
				originalPackageID,
			)
		}

		if savedOriginal.TotalClasses != originalTotal {
			t.Fatalf(
				"original TotalClasses = %d, want %d",
				savedOriginal.TotalClasses,
				originalTotal,
			)
		}

		if savedOriginal.ClassesRemaining != originalRemaining {
			t.Fatalf(
				"original ClassesRemaining = %d, want %d",
				savedOriginal.ClassesRemaining,
				originalRemaining,
			)
		}

		if savedOriginal.Contract != originalContract {
			t.Fatalf(
				"original Contract = %q, want %q",
				savedOriginal.Contract,
				originalContract,
			)
		}

		if !savedOriginal.Active {
			t.Fatal("original enrollment became inactive")
		}
	})

	t.Run("rejects inactive original enrollment", func(t *testing.T) {
		db := setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		original := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := original.Create(); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if err := db.Model(&Enrollment{}).
			Where("id = ?", original.ID).
			Update("active", false).Error; err != nil {
			t.Fatalf("failed to deactivate original: %v", err)
		}

		var renewed *Enrollment
		err := db.Transaction(func(tx *gorm.DB) error {
			var err error
			renewed, err = original.RenewWithTx(
				tx,
				course.ID,
				pkg.ID,
				"renewed-contract.pdf",
			)
			return err
		})

		if !errors.Is(err, ErrEnrollmentRenewalInactive) {
			t.Fatalf(
				"RenewWithTx() error = %v, want %v",
				err,
				ErrEnrollmentRenewalInactive,
			)
		}

		if renewed != nil {
			t.Fatal("expected no renewed enrollment")
		}
	})

	t.Run("rejects nonexistent original enrollment", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		original := Enrollment{
			Model: uadmin.Model{
				ID: 99999,
			},
		}

		var renewed *Enrollment
		err := uadmin.GetDB().Transaction(func(tx *gorm.DB) error {
			var err error
			renewed, err = original.RenewWithTx(
				tx,
				course.ID,
				pkg.ID,
				"renewed-contract.pdf",
			)
			return err
		})

		if !errors.Is(err, ErrEnrollmentRenewalNotFound) {
			t.Fatalf(
				"RenewWithTx() error = %v, want %v",
				err,
				ErrEnrollmentRenewalNotFound,
			)
		}

		if renewed != nil {
			t.Fatal("expected no renewed enrollment")
		}
	})

	t.Run("rejects duplicate active course enrollment", func(t *testing.T) {
		setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		course := createEnrollmentTestCourse(t)
		pkg := createEnrollmentTestPackage(t, 10, 0)

		original := Enrollment{
			StudentID: student.ID,
			CourseID:  course.ID,
			PackageID: pkg.ID,
		}

		if err := original.Create(); err != nil {
			t.Fatalf("original Create() error = %v", err)
		}

		renewalPackage := createEnrollmentTestPackage(t, 20, 0)

		var renewed *Enrollment
		err := uadmin.GetDB().Transaction(func(tx *gorm.DB) error {
			var err error
			renewed, err = original.RenewWithTx(
				tx,
				course.ID,
				renewalPackage.ID,
				"renewed-contract.pdf",
			)
			return err
		})

		if !errors.Is(err, ErrEnrollmentAlreadyExists) {
			t.Fatalf(
				"RenewWithTx() error = %v, want %v",
				err,
				ErrEnrollmentAlreadyExists,
			)
		}

		if renewed != nil {
			t.Fatal("expected no renewed enrollment")
		}
	})

	t.Run("allows renewal with a different course", func(t *testing.T) {
		db := setupEnrollmentCreateTestDB(t)

		student := createEnrollmentTestStudent(t)
		originalCourse := createEnrollmentTestCourse(t)

		newCourse := Course{
			Title:  "Business English",
			Active: true,
		}

		if err := uadmin.Save(&newCourse); err != nil {
			t.Fatalf("failed to create renewal course: %v", err)
		}

		originalPackage := createEnrollmentTestPackage(t, 10, 0)
		renewalPackage := createEnrollmentTestPackage(t, 20, 5)

		original := Enrollment{
			StudentID: student.ID,
			CourseID:  originalCourse.ID,
			PackageID: originalPackage.ID,
		}

		if err := original.Create(); err != nil {
			t.Fatalf("original Create() error = %v", err)
		}

		var renewed *Enrollment

		err := db.Transaction(func(tx *gorm.DB) error {
			var err error

			renewed, err = original.RenewWithTx(
				tx,
				newCourse.ID,
				renewalPackage.ID,
				"",
			)

			return err
		})

		if err != nil {
			t.Fatalf("RenewWithTx() error = %v", err)
		}

		if renewed == nil {
			t.Fatal("expected renewed enrollment")
		}

		if err != nil {
			t.Fatalf("RenewWithTx() error = %v", err)
		}

		if renewed == nil {
			t.Fatal("expected renewed enrollment")
		}

		if renewed.CourseID != newCourse.ID {
			t.Fatalf(
				"renewed CourseID = %d, want %d",
				renewed.CourseID,
				newCourse.ID,
			)
		}

		if renewed.TotalClasses != 25 {
			t.Fatalf(
				"renewed TotalClasses = %d, want 25",
				renewed.TotalClasses,
			)
		}

		if renewed.ClassesRemaining != 25 {
			t.Fatalf(
				"renewed ClassesRemaining = %d, want 25",
				renewed.ClassesRemaining,
			)
		}

		var count int64

		if err := db.Model(&Enrollment{}).
			Where(
				"student_id = ? AND course_id = ? AND active = ?",
				student.ID,
				newCourse.ID,
				true,
			).
			Count(&count).Error; err != nil {
			t.Fatalf("failed to count renewed enrollments: %v", err)
		}

		if count != 1 {
			t.Fatalf(
				"active renewed enrollment count = %d, want 1",
				count,
			)
		}
	})
}
