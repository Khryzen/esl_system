package models

import (
	"errors"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrClassEnrollmentRequired = errors.New(
		"Enrollment is required.",
	)

	ErrClassEnrollmentNotFound = errors.New(
		"The selected enrollment could not be found.",
	)

	ErrClassEnrollmentInactive = errors.New(
		"The selected enrollment is no longer active.",
	)

	ErrClassNoCreditsRemaining = errors.New(
		"The selected enrollment has no classes remaining.",
	)

	ErrClassScheduleFailed = errors.New(
		"The class could not be scheduled. Please try again.",
	)

	ErrClassAlreadyCancelled = errors.New(
		"The class has already been cancelled.",
	)

	ErrClassNotCancelled = errors.New(
		"The class must be cancelled before its credit can be refunded.",
	)

	ErrClassCreditAlreadyRefunded = errors.New(
		"The class credit has already been refunded.",
	)

	ErrClassCancelFailed = errors.New(
		"The class could not be cancelled. Please try again.",
	)

	ErrClassCreditRefundFailed = errors.New(
		"The class credit could not be refunded. Please try again.",
	)

	ErrClassNotFound = errors.New(
		"The selected class could not be found.",
	)

	ErrClassRescheduleFailed = errors.New(
		"The class could not be rescheduled. Please try again.",
	)

	ErrClassRescheduleCourseChanged = errors.New(
		"The class cannot be rescheduled because the enrollment course has changed.",
	)

	ErrClassAttendanceRecorded = errors.New(
		"The class attendance has already been recorded.",
	)

	ErrClassReplacementAlreadyExists = errors.New(
		"The replacement class must not already exist.",
	)

	ErrClassDateInPast = errors.New(
		"The class date cannot be in the past.",
	)

	ErrClassStartTimeRequired = errors.New(
		"The class start time is required.",
	)

	ErrClassStartTimeInPast = errors.New(
		"The class start time cannot be in the past.",
	)

	ErrClassScheduleConflict = errors.New(
		"The selected class time conflicts with another scheduled class.",
	)

	ErrClassAlreadyScheduled = errors.New(
		"The class has already been scheduled.",
	)

	ErrClassDateRequired = errors.New(
		"The class date is required.",
	)
)

type Class struct {
	uadmin.Model
	ClassDate time.Time
	StartTime *time.Time
	EndTime   *time.Time

	Enrollment   Enrollment
	EnrollmentID uint

	Course   Course
	CourseID uint

	Student   Student
	StudentID uint

	Present        bool
	Absent         bool
	Cancelled      bool
	CreditRefunded bool

	RescheduledFromID uint
	RescheduledFrom   *Class
}

func (c *Class) Schedule() error {
	if c.EnrollmentID == 0 {
		return ErrClassEnrollmentRequired
	}

	if c.ID != 0 {
		return ErrClassAlreadyScheduled
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var enrollment Enrollment

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&enrollment, c.EnrollmentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassEnrollmentNotFound
			}

			return err
		}

		if !enrollment.Active {
			return ErrClassEnrollmentInactive
		}

		if enrollment.ClassesRemaining <= 0 {
			return ErrClassNoCreditsRemaining
		}

		var packageModel Package

		if err := tx.
			First(&packageModel, enrollment.PackageID).Error; err != nil {
			return err
		}

		now := time.Now().In(time.Local)

		start, end, err := validateClassSchedule(
			c.ClassDate,
			c.StartTime,
			packageModel.ClassDurationInMinutes,
			now,
		)
		if err != nil {
			return err
		}

		conflict, err := hasClassScheduleConflict(
			tx,
			start,
			end,
			0,
		)
		if err != nil {
			return err
		}

		if conflict {
			return ErrClassScheduleConflict
		}

		c.StudentID = enrollment.StudentID
		c.CourseID = enrollment.CourseID
		c.StartTime = &start
		c.EndTime = &end
		c.Cancelled = false
		c.CreditRefunded = false
		c.Present = false
		c.Absent = false

		if err := tx.Create(c).Error; err != nil {
			return err
		}

		enrollment.ClassesRemaining--

		if enrollment.ClassesRemaining == 0 {
			enrollment.Active = false
		}

		if err := tx.Save(&enrollment).Error; err != nil {
			return err
		}

		c.Enrollment = enrollment

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrClassEnrollmentRequired),
		errors.Is(err, ErrClassEnrollmentNotFound),
		errors.Is(err, ErrClassEnrollmentInactive),
		errors.Is(err, ErrClassNoCreditsRemaining),
		errors.Is(err, ErrClassDateInPast),
		errors.Is(err, ErrClassStartTimeRequired),
		errors.Is(err, ErrClassStartTimeInPast),
		errors.Is(err, ErrPackageInvalidClassDuration),
		errors.Is(err, ErrClassScheduleConflict):
		return err
	}

	return errors.Join(ErrClassScheduleFailed, err)
}

func (c *Class) Cancel() error {
	if c.ID == 0 {
		return ErrClassNotFound
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var class Class

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&class, c.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassNotFound
			}

			return err
		}

		if class.Cancelled {
			return ErrClassAlreadyCancelled
		}

		if class.Present || class.Absent {
			return ErrClassAttendanceRecorded
		}

		class.Cancelled = true

		if err := tx.Save(&class).Error; err != nil {
			return err
		}

		c.Cancelled = class.Cancelled

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrClassNotFound),
		errors.Is(err, ErrClassAlreadyCancelled),
		errors.Is(err, ErrClassAttendanceRecorded):
		return err
	}

	return errors.Join(ErrClassCancelFailed, err)
}

func (c *Class) RefundCredit() error {
	if c.ID == 0 {
		return ErrClassNotFound
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var class Class

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&class, c.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassNotFound
			}

			return err
		}

		if !class.Cancelled {
			return ErrClassNotCancelled
		}

		if class.CreditRefunded {
			return ErrClassCreditAlreadyRefunded
		}

		if class.EnrollmentID == 0 {
			return ErrClassEnrollmentNotFound
		}

		var enrollment Enrollment

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&enrollment, class.EnrollmentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassEnrollmentNotFound
			}

			return err
		}

		enrollment.ClassesRemaining++

		if enrollment.ClassesRemaining > 0 {
			enrollment.Active = true
		}

		class.CreditRefunded = true

		if err := tx.Save(&enrollment).Error; err != nil {
			return err
		}

		if err := tx.Save(&class).Error; err != nil {
			return err
		}

		c.CreditRefunded = class.CreditRefunded

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrClassNotFound),
		errors.Is(err, ErrClassNotCancelled),
		errors.Is(err, ErrClassCreditAlreadyRefunded),
		errors.Is(err, ErrClassEnrollmentNotFound):
		return err
	}

	return errors.Join(ErrClassCreditRefundFailed, err)
}

func (c *Class) Reschedule(replacement *Class) error {
	if c.ID == 0 {
		return ErrClassNotFound
	}

	if replacement == nil {
		return ErrClassRescheduleFailed
	}

	if replacement.ID != 0 {
		return ErrClassReplacementAlreadyExists
	}

	db := uadmin.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		var original Class

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&original, c.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassNotFound
			}

			return err
		}

		if original.Cancelled {
			return ErrClassAlreadyCancelled
		}

		if original.Present || original.Absent {
			return ErrClassAttendanceRecorded
		}

		if original.EnrollmentID == 0 {
			return ErrClassEnrollmentNotFound
		}

		var enrollment Enrollment

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&enrollment, original.EnrollmentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClassEnrollmentNotFound
			}

			return err
		}

		if original.CourseID != enrollment.CourseID {
			return ErrClassRescheduleCourseChanged
		}

		var packageModel Package

		if err := tx.
			First(&packageModel, enrollment.PackageID).Error; err != nil {
			return err
		}

		now := time.Now().In(time.Local)

		start, end, err := validateClassSchedule(
			replacement.ClassDate,
			replacement.StartTime,
			packageModel.ClassDurationInMinutes,
			now,
		)
		if err != nil {
			return err
		}

		// Exclude the original class from the conflict check.
		// It is still present in the database at this point,
		// but it will be cancelled as part of this transaction.
		conflict, err := hasClassScheduleConflict(
			tx,
			start,
			end,
			original.ID,
		)
		if err != nil {
			return err
		}

		if conflict {
			return ErrClassScheduleConflict
		}

		// Refund the original class credit.
		// The refunded credit will immediately be consumed
		// by the replacement class, so the enrollment's
		// final credit balance remains unchanged.
		enrollment.ClassesRemaining++

		if enrollment.ClassesRemaining > 0 {
			enrollment.Active = true
		}

		original.Cancelled = true
		original.CreditRefunded = true

		// The replacement inherits the original enrollment,
		// student, and course.
		// Only the replacement date and start time are supplied
		// by the caller. EndTime is calculated from the package.
		replacement.EnrollmentID = original.EnrollmentID
		replacement.StudentID = enrollment.StudentID
		replacement.CourseID = enrollment.CourseID
		replacement.RescheduledFromID = original.ID
		replacement.StartTime = &start
		replacement.EndTime = &end
		replacement.Cancelled = false
		replacement.CreditRefunded = false
		replacement.Present = false
		replacement.Absent = false

		if err := tx.Save(&original).Error; err != nil {
			return err
		}

		if err := tx.Save(&enrollment).Error; err != nil {
			return err
		}

		if err := tx.Create(replacement).Error; err != nil {
			return err
		}

		// The replacement consumes the credit that was
		// just refunded.
		enrollment.ClassesRemaining--

		if enrollment.ClassesRemaining == 0 {
			enrollment.Active = false
		}

		if err := tx.Save(&enrollment).Error; err != nil {
			return err
		}

		c.Cancelled = original.Cancelled
		c.CreditRefunded = original.CreditRefunded

		return nil
	})

	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrClassNotFound),
		errors.Is(err, ErrClassAlreadyCancelled),
		errors.Is(err, ErrClassAttendanceRecorded),
		errors.Is(err, ErrClassEnrollmentNotFound),
		errors.Is(err, ErrClassRescheduleCourseChanged),
		errors.Is(err, ErrClassDateInPast),
		errors.Is(err, ErrClassStartTimeRequired),
		errors.Is(err, ErrClassStartTimeInPast),
		errors.Is(err, ErrPackageInvalidClassDuration),
		errors.Is(err, ErrClassScheduleConflict):
		return err
	}

	return errors.Join(ErrClassRescheduleFailed, err)
}

func classDateTime(date time.Time, timeOfDay time.Time) time.Time {
	year, month, day := date.In(time.Local).Date()

	return time.Date(
		year,
		month,
		day,
		timeOfDay.In(time.Local).Hour(),
		timeOfDay.In(time.Local).Minute(),
		timeOfDay.In(time.Local).Second(),
		timeOfDay.In(time.Local).Nanosecond(),
		time.Local,
	)
}

func validateClassSchedule(
	classDate time.Time,
	startTime *time.Time,
	duration int,
	now time.Time,
) (time.Time, time.Time, error) {

	if startTime == nil {
		return time.Time{}, time.Time{}, ErrClassStartTimeRequired
	}

	if classDate.IsZero() {
		return time.Time{}, time.Time{}, ErrClassDateRequired
	}

	if duration <= 0 {
		return time.Time{}, time.Time{}, ErrPackageInvalidClassDuration
	}
	year, month, day := classDate.In(time.Local).Date()

	today := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		now.Location(),
	)

	scheduledDate := time.Date(
		year,
		month,
		day,
		0,
		0,
		0,
		0,
		now.Location(),
	)

	if scheduledDate.Before(today) {
		return time.Time{}, time.Time{}, ErrClassDateInPast
	}

	start := classDateTime(classDate, *startTime)

	if scheduledDate.Equal(today) && start.Before(now) {
		return time.Time{}, time.Time{}, ErrClassStartTimeInPast
	}

	end := start.Add(time.Duration(duration) * time.Minute)

	return start, end, nil
}

func hasClassScheduleConflict(
	tx *gorm.DB,
	start time.Time,
	end time.Time,
	excludeID uint,
) (bool, error) {
	var count int64

	query := tx.
		Model(&Class{}).
		Where("cancelled = ?", false).
		Where("start_time < ?", end).
		Where("end_time > ?", start)

	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
