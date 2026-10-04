package views

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

const (
	classCalendarDateLayout     = "2006-01-02"
	classCalendarDateTimeLayout = "2006-01-02 15:04"
	classCalendarJSONTimeLayout = "2006-01-02T15:04:05"
)

type classUserError string

func (e classUserError) Error() string {
	return string(e)
}

type classCalendarAssessment struct {
	ID                 uint    `json:"id"`
	Rating             float64 `json:"rating"`
	GrammarCorrections string  `json:"grammar_corrections"`
	Recommendation     string  `json:"recommendation"`
	Homework           string  `json:"homework"`
	Remarks            string  `json:"remarks"`
	HomeworkTitle      string  `json:"homework_title"`
}

type classCalendarClass struct {
	ID              uint                     `json:"id"`
	Student         string                   `json:"student"`
	Course          string                   `json:"course"`
	Package         string                   `json:"package"`
	Reference       string                   `json:"reference"`
	EnrollmentID    uint                     `json:"enrollment_id"`
	Start           string                   `json:"start"`
	End             string                   `json:"end"`
	DurationMinutes int                      `json:"duration_minutes"`
	Status          string                   `json:"status"`
	CreditRefunded  bool                     `json:"credit_refunded"`
	Cancelled       bool                     `json:"cancelled"`
	Assessment      *classCalendarAssessment `json:"assessment"`
}

type classCalendarEnrollment struct {
	ID               uint   `json:"id"`
	Student          string `json:"student"`
	Course           string `json:"course"`
	Package          string `json:"package"`
	DurationMinutes  int    `json:"duration_minutes"`
	ClassesRemaining int    `json:"classes_remaining"`
}

func ClassHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}

	switch {
	case r.Method == http.MethodPost && r.URL.Query().Get("start") != "":
		classSchedule(w, r)
		return context

	case r.Method == http.MethodPost:
		switch r.FormValue("action") {
		case "set_attendance":
			classSetAttendance(w, r)

		case "save_feedback":
			classSaveFeedback(w, r)

		case "cancel":
			classCancel(w, r)

		case "refund_credit":
			classRefundCredit(w, r)

		case "reschedule":
			classReschedule(w, r)

		default:
			classCreate(w, r)
		}

		return context
	}

	return classPageContext()
}

func classPageContext() map[string]interface{} {
	return map[string]interface{}{
		"Title": "Classes",
	}
}

func classSchedule(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	from, err := time.ParseInLocation(
		classCalendarDateLayout,
		query.Get("start"),
		time.Local,
	)

	if err != nil {
		classFail(
			w,
			r,
			classUserError("Invalid start date."),
		)
		return
	}

	days, err := strconv.Atoi(query.Get("days"))

	if err != nil || days < 1 || days > 62 {
		days = 1
	}

	to := from.AddDate(0, 0, days)

	students := []models.Student{}
	uadmin.All(&students)

	studentNames := map[uint]string{}

	for _, student := range students {
		studentNames[student.ID] = strings.TrimSpace(
			student.FirstName + " " + student.LastName,
		)
	}

	courses := []models.Course{}
	uadmin.All(&courses)

	courseTitles := map[uint]string{}

	for _, course := range courses {
		courseTitles[course.ID] = course.Title
	}

	packages := []models.Package{}
	uadmin.All(&packages)

	packageByID := map[uint]models.Package{}

	for _, pkg := range packages {
		packageByID[pkg.ID] = pkg
	}

	enrollments := []models.Enrollment{}
	uadmin.All(&enrollments)

	enrollmentByID := map[uint]models.Enrollment{}
	choices := []classCalendarEnrollment{}

	for _, enrollment := range enrollments {
		enrollmentByID[enrollment.ID] = enrollment

		if !enrollment.Active || enrollment.ClassesRemaining < 1 {
			continue
		}

		pkg := packageByID[enrollment.PackageID]

		choices = append(
			choices,
			classCalendarEnrollment{
				ID:               enrollment.ID,
				Student:          studentNames[enrollment.StudentID],
				Course:           courseTitles[enrollment.CourseID],
				Package:          pkg.Name,
				DurationMinutes:  pkg.ClassDurationInMinutes,
				ClassesRemaining: enrollment.ClassesRemaining,
			},
		)
	}

	classes := []models.Class{}

	uadmin.Filter(
		&classes,
		"start_time >= ? AND start_time < ?",
		from,
		to,
	)

	if query.Get("view") == "month" {
		counts := map[string]int{}

		for _, class := range classes {
			if class.StartTime == nil {
				continue
			}

			date := class.StartTime.
				In(time.Local).
				Format(classCalendarDateLayout)

			counts[date]++
		}

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status": "ok",
			"view":   "month",
			"counts": counts,
		})

		return
	}

	assessments := []models.Assessment{}
	uadmin.All(&assessments)

	assessmentByClass := map[uint]models.Assessment{}

	for _, assessment := range assessments {
		assessmentByClass[assessment.ClassID] = assessment
	}

	homeworks := []models.Homework{}
	uadmin.All(&homeworks)

	homeworkByAssessment := map[uint]models.Homework{}

	for _, homework := range homeworks {
		homeworkByAssessment[homework.AssessmentID] = homework
	}

	result := []classCalendarClass{}

	for _, class := range classes {
		if class.StartTime == nil || class.EndTime == nil {
			continue
		}

		enrollment := enrollmentByID[class.EnrollmentID]
		pkg := packageByID[enrollment.PackageID]

		status := ""

		if class.Present {
			status = "present"
		} else if class.Absent {
			status = "absent"
		}

		var assessment *classCalendarAssessment

		if value, ok := assessmentByClass[class.ID]; ok {
			homework := homeworkByAssessment[value.ID]

			assessment = &classCalendarAssessment{
				ID:                 value.ID,
				Rating:             value.Rating,
				GrammarCorrections: value.GrammarCorrections,
				Recommendation:     value.Recommendation,
				Homework:           value.Homework,
				Remarks:            value.Remarks,
				HomeworkTitle:      homework.Title,
			}
		}

		result = append(
			result,
			classCalendarClass{
				ID:              class.ID,
				Student:         studentNames[class.StudentID],
				Course:          courseTitles[enrollment.CourseID],
				Package:         pkg.Name,
				Reference:       enrollment.ReferenceNumber,
				EnrollmentID:    class.EnrollmentID,
				Start:           class.StartTime.In(time.Local).Format(classCalendarJSONTimeLayout),
				End:             class.EndTime.In(time.Local).Format(classCalendarJSONTimeLayout),
				DurationMinutes: int(class.EndTime.Sub(*class.StartTime).Minutes()),
				Status:          status,
				CreditRefunded:  class.CreditRefunded,
				Cancelled:       class.Cancelled,
				Assessment:      assessment,
			},
		)
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":      "ok",
		"view":        "day",
		"classes":     result,
		"enrollments": choices,
	})
}

func classCreate(w http.ResponseWriter, r *http.Request) {
	enrollmentID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("enrollment_id")),
		10,
		64,
	)

	if err != nil || enrollmentID == 0 {
		classFail(
			w,
			r,
			classUserError("Select an enrollment."),
		)
		return
	}

	start, err := time.ParseInLocation(
		classCalendarDateTimeLayout,
		strings.TrimSpace(r.FormValue("date"))+" "+
			strings.TrimSpace(r.FormValue("start_time")),
		time.Local,
	)

	if err != nil {
		classFail(
			w,
			r,
			classUserError(
				"Enter a valid date and start time.",
			),
		)
		return
	}

	class := models.Class{
		ClassDate: time.Date(
			start.Year(),
			start.Month(),
			start.Day(),
			0,
			0,
			0,
			0,
			time.Local,
		),
		StartTime:    &start,
		EnrollmentID: uint(enrollmentID),
	}

	if err := class.Schedule(); err != nil {
		classFail(w, r, classScheduleError(err))
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":            "ok",
		"class_id":          class.ID,
		"classes_remaining": class.Enrollment.ClassesRemaining,
	})
}

func classScheduleError(err error) error {
	switch {
	case errors.Is(err, models.ErrClassEnrollmentRequired),
		errors.Is(err, models.ErrClassEnrollmentNotFound):
		return classUserError("Enrollment not found.")

	case errors.Is(err, models.ErrClassEnrollmentInactive):
		return classUserError(
			"This enrollment is not active.",
		)

	case errors.Is(err, models.ErrClassNoCreditsRemaining):
		return classUserError(
			"This enrollment has no classes remaining.",
		)

	case errors.Is(err, models.ErrClassDateInPast):
		return classUserError(
			"The class date cannot be in the past.",
		)

	case errors.Is(err, models.ErrClassStartTimeRequired),
		errors.Is(err, models.ErrClassStartTimeInPast):
		return classUserError(
			"Enter a valid future start time.",
		)

	case errors.Is(err, models.ErrPackageInvalidClassDuration):
		return classUserError(
			"The package has no class duration set.",
		)

	case errors.Is(err, models.ErrClassScheduleConflict):
		return classUserError(
			"Another class is already scheduled during that time.",
		)

	default:
		return err
	}
}

func classSetAttendance(w http.ResponseWriter, r *http.Request) {
	classID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("class_id")),
		10,
		64,
	)

	if err != nil || classID == 0 {
		classFail(
			w,
			r,
			classUserError("Invalid class."),
		)
		return
	}

	status := strings.TrimSpace(r.FormValue("status"))

	if status != "present" && status != "absent" {
		classFail(
			w,
			r,
			classUserError("Choose Present or Absent."),
		)
		return
	}

	refund := status == "absent" &&
		r.FormValue("refund") == "1"

	class := models.Class{
		Model: uadmin.Model{
			ID: uint(classID),
		},
	}

	if err := class.SetAttendance(
		status == "present",
		refund,
	); err != nil {
		classFail(w, r, classAttendanceError(err))
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":          "ok",
		"class_status":    status,
		"credit_refunded": class.CreditRefunded,
	})
}

func classAttendanceError(err error) error {
	switch {
	case errors.Is(err, models.ErrClassNotFound):
		return classUserError("Class not found.")

	case errors.Is(err, models.ErrClassAttendanceRecorded):
		return classUserError(
			"This class has already been tagged.",
		)

	case errors.Is(err, models.ErrClassEnrollmentNotFound):
		return classUserError(
			"The enrollment for this class could not be found.",
		)

	case errors.Is(err, models.ErrClassCreditsAlreadyFull):
		return classUserError(
			"The enrollment already has all of its classes available.",
		)

	case errors.Is(err, models.ErrClassPresentCannotRefund):
		return classUserError(
			"A present class cannot have its credit refunded.",
		)

	default:
		return err
	}
}

func classCancel(w http.ResponseWriter, r *http.Request) {
	classID, err := parseClassID(r)

	if err != nil {
		classFail(w, r, err)
		return
	}

	class := models.Class{
		Model: uadmin.Model{
			ID: classID,
		},
	}

	if err := class.Cancel(); err != nil {
		switch {
		case errors.Is(err, models.ErrClassNotFound):
			classFail(
				w,
				r,
				classUserError("Class not found."),
			)

		case errors.Is(err, models.ErrClassAlreadyCancelled):
			classFail(
				w,
				r,
				classUserError("This class is already cancelled."),
			)

		case errors.Is(err, models.ErrClassAttendanceRecorded):
			classFail(
				w,
				r,
				classUserError(
					"Attendance has already been recorded for this class.",
				),
			)

		default:
			classFail(w, r, err)
		}

		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status": "ok",
	})
}

func classRefundCredit(w http.ResponseWriter, r *http.Request) {
	classID, err := parseClassID(r)

	if err != nil {
		classFail(w, r, err)
		return
	}

	class := models.Class{
		Model: uadmin.Model{
			ID: classID,
		},
	}

	if err := class.RefundCredit(); err != nil {
		switch {
		case errors.Is(err, models.ErrClassNotFound):
			classFail(
				w,
				r,
				classUserError("Class not found."),
			)

		case errors.Is(err, models.ErrClassNotCancelled):
			classFail(
				w,
				r,
				classUserError(
					"The class must be cancelled before its credit can be refunded.",
				),
			)

		case errors.Is(err, models.ErrClassCreditAlreadyRefunded):
			classFail(
				w,
				r,
				classUserError(
					"The class credit has already been refunded.",
				),
			)

		case errors.Is(err, models.ErrClassCreditsAlreadyFull):
			classFail(
				w,
				r,
				classUserError(
					"The enrollment already has all of its classes available.",
				),
			)

		case errors.Is(err, models.ErrClassEnrollmentNotFound):
			classFail(
				w,
				r,
				classUserError(
					"The enrollment for this class could not be found.",
				),
			)

		default:
			classFail(w, r, err)
		}

		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":          "ok",
		"credit_refunded": class.CreditRefunded,
	})
}

func classReschedule(w http.ResponseWriter, r *http.Request) {
	classID, err := parseClassID(r)

	if err != nil {
		classFail(w, r, err)
		return
	}

	date := strings.TrimSpace(r.FormValue("date"))
	startTime := strings.TrimSpace(r.FormValue("start_time"))

	start, err := time.ParseInLocation(
		classCalendarDateTimeLayout,
		date+" "+startTime,
		time.Local,
	)

	if err != nil {
		classFail(
			w,
			r,
			classUserError(
				"Enter a valid date and start time.",
			),
		)
		return
	}

	replacement := &models.Class{
		ClassDate: time.Date(
			start.Year(),
			start.Month(),
			start.Day(),
			0,
			0,
			0,
			0,
			time.Local,
		),
		StartTime: &start,
	}

	class := models.Class{
		Model: uadmin.Model{
			ID: classID,
		},
	}

	if err := class.Reschedule(replacement); err != nil {
		classFail(w, r, classRescheduleError(err))
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":            "ok",
		"class_id":          replacement.ID,
		"replacement_id":    replacement.ID,
		"original_class_id": class.ID,
	})
}

func classRescheduleError(err error) error {
	switch {
	case errors.Is(err, models.ErrClassNotFound):
		return classUserError("Class not found.")

	case errors.Is(err, models.ErrClassAlreadyCancelled):
		return classUserError(
			"This class is already cancelled.",
		)

	case errors.Is(err, models.ErrClassAttendanceRecorded):
		return classUserError(
			"Attendance has already been recorded for this class.",
		)

	case errors.Is(err, models.ErrClassEnrollmentNotFound):
		return classUserError(
			"The enrollment for this class could not be found.",
		)

	case errors.Is(err, models.ErrClassRescheduleCourseChanged):
		return classUserError(
			"The class cannot be rescheduled because the enrollment course has changed.",
		)

	case errors.Is(err, models.ErrClassDateInPast):
		return classUserError(
			"The class date cannot be in the past.",
		)

	case errors.Is(err, models.ErrClassStartTimeRequired),
		errors.Is(err, models.ErrClassStartTimeInPast):
		return classUserError(
			"Enter a valid future start time.",
		)

	case errors.Is(err, models.ErrPackageInvalidClassDuration):
		return classUserError(
			"The package has no class duration set.",
		)

	case errors.Is(err, models.ErrClassScheduleConflict):
		return classUserError(
			"Another class is already scheduled during that time.",
		)

	default:
		return err
	}
}

func classSaveFeedback(w http.ResponseWriter, r *http.Request) {
	classID, err := parseClassID(r)

	if err != nil {
		classFail(w, r, err)
		return
	}

	rating, err := strconv.ParseFloat(
		strings.TrimSpace(r.FormValue("rating")),
		64,
	)

	if err != nil || rating < 0 || rating > 10 {
		classFail(
			w,
			r,
			classUserError(
				"Enter a rating between 0 and 10.",
			),
		)
		return
	}

	db := uadmin.GetDB()

	var assessmentID uint

	err = db.Transaction(func(tx *gorm.DB) error {
		var class models.Class

		if err := tx.First(&class, classID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return classUserError("Class not found.")
			}

			return err
		}

		if !class.Present {
			return classUserError(
				"Feedback can only be given for a class marked present.",
			)
		}

		var assessment models.Assessment

		err := tx.
			Where("class_id = ?", class.ID).
			First(&assessment).
			Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			assessment = models.Assessment{
				ClassID: class.ID,
				Date:    time.Now(),
			}
		} else if err != nil {
			return err
		}

		assessment.Rating = rating
		assessment.GrammarCorrections =
			r.FormValue("grammar_corrections")
		assessment.Recommendation =
			r.FormValue("recommendation")
		assessment.Homework =
			r.FormValue("homework")
		assessment.Remarks =
			r.FormValue("remarks")

		if err := tx.Save(&assessment).Error; err != nil {
			return err
		}

		assessmentID = assessment.ID

		title := strings.TrimSpace(
			r.FormValue("homework_title"),
		)

		if title == "" {
			return nil
		}

		var homework models.Homework

		err = tx.
			Where("assessment_id = ?", assessment.ID).
			First(&homework).
			Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			homework = models.Homework{
				AssessmentID: assessment.ID,
			}
		} else if err != nil {
			return err
		}

		homework.Title = title

		if err := tx.Save(&homework).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		classFail(w, r, err)
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":        "ok",
		"assessment_id": assessmentID,
	})
}

func parseClassID(r *http.Request) (uint, error) {
	classID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("class_id")),
		10,
		64,
	)

	if err != nil || classID == 0 {
		return 0, classUserError("Invalid class.")
	}

	return uint(classID), nil
}

func classFail(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	message := "Something went wrong while processing the class."

	var userErr classUserError

	if errors.As(err, &userErr) {
		message = userErr.Error()
	} else {
		uadmin.Trail(
			uadmin.ERROR,
			"ClassHandler: %v",
			err,
		)
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}
