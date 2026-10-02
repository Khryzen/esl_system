package views

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

const (
	calendarDateLayout     = "2006-01-02"
	calendarDateTimeLayout = "2006-01-02 15:04"
	calendarJSONTimeLayout = "2006-01-02T15:04:05"

	// An active enrollment at or below this many remaining classes shows up in the
	// "Renewals Needed" panel.
	lowBalanceThreshold = 2

	// Each dashboard sidebar panel shows at most this many rows; the rest are
	// summarized as "+N more" rather than growing the page unbounded.
	summaryListLimit = 8
)

type dashboardUserError string

func (e dashboardUserError) Error() string {
	return string(e)
}

// renewalRow is one row in the "Renewals Needed" sidebar panel.
type renewalRow struct {
	ID               uint
	ReferenceNumber  string
	Student          string
	Course           string
	Package          string
	ClassesRemaining int
}

// followUpRow is one row in the "Attendance Follow-up" sidebar panel.
// DateISO and ID are read by dashboard.js (data-followup-date /
// data-followup-class) to jump the calendar to that class's day and open
// its detail modal.
type followUpRow struct {
	ID      uint
	Student string
	Course  string
	Date    string
	DateISO string
	Time    string
}

type calendarAssessment struct {
	ID                 uint    `json:"id"`
	Rating             float64 `json:"rating"`
	GrammarCorrections string  `json:"grammar_corrections"`
	Recommendation     string  `json:"recommendation"`
	Homework           string  `json:"homework"`
	Remarks            string  `json:"remarks"`
	HomeworkTitle      string  `json:"homework_title"`
}

type calendarClass struct {
	ID              uint                `json:"id"`
	Student         string              `json:"student"`
	Course          string              `json:"course"`
	Package         string              `json:"package"`
	Reference       string              `json:"reference"`
	EnrollmentID    uint                `json:"enrollment_id"`
	Start           string              `json:"start"`
	End             string              `json:"end"`
	DurationMinutes int                 `json:"duration_minutes"`
	Status          string              `json:"status"` // "", "present", "absent"
	CreditRefunded  bool                `json:"credit_refunded"`
	Assessment      *calendarAssessment `json:"assessment"`
}

type calendarEnrollment struct {
	ID               uint   `json:"id"`
	Student          string `json:"student"`
	Course           string `json:"course"`
	Package          string `json:"package"`
	DurationMinutes  int    `json:"duration_minutes"`
	ClassesRemaining int    `json:"classes_remaining"`
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}

	switch {
	case r.Method == http.MethodPost && r.URL.Query().Get("start") != "":
		sendSchedule(w, r)
		return context

	case r.Method == http.MethodPost:
		switch r.FormValue("action") {
		case "set_attendance":
			setAttendance(w, r)

		case "save_feedback":
			saveFeedback(w, r)

		default:
			createClass(w, r)
		}

		return context
	}

	students := []models.Student{}
	uadmin.All(&students)

	courses := []models.Course{}
	uadmin.Filter(&courses, "active = ?", true)

	packages := []models.Package{}
	uadmin.Filter(&packages, "active = ?", true)

	context["NumberOfStudents"] = len(students)
	context["NumberOfCourses"] = len(courses)
	context["NumberOfPackages"] = len(packages)

	renewals, renewalsMore := renewalsNeeded()
	context["Renewals"] = renewals
	context["RenewalsTotal"] = len(renewals) + renewalsMore
	context["RenewalsMore"] = renewalsMore

	followUps, followUpsMore := attendanceFollowUps()
	context["AttendanceFollowUps"] = followUps
	context["AttendanceFollowUpsTotal"] = len(followUps) + followUpsMore
	context["AttendanceFollowUpsMore"] = followUpsMore

	return context
}

// nameLookups returns quick id->name maps for students and courses, and id->Package
// for packages, shared by the dashboard's summary panels.
func nameLookups() (
	students map[uint]string,
	courses map[uint]string,
	packages map[uint]models.Package,
) {
	studentRows := []models.Student{}
	uadmin.All(&studentRows)

	students = map[uint]string{}

	for _, s := range studentRows {
		students[s.ID] = strings.TrimSpace(
			s.FirstName + " " + s.LastName,
		)
	}

	courseRows := []models.Course{}
	uadmin.All(&courseRows)

	courses = map[uint]string{}

	for _, c := range courseRows {
		courses[c.ID] = c.Title
	}

	packageRows := []models.Package{}
	uadmin.All(&packageRows)

	packages = map[uint]models.Package{}

	for _, p := range packageRows {
		packages[p.ID] = p
	}

	return
}

// renewalsNeeded returns active enrollments at or below lowBalanceThreshold
// classes remaining, most urgent (fewest classes left) first, capped at
// summaryListLimit. The second return value is how many more rows exist
// beyond that cap.
func renewalsNeeded() ([]renewalRow, int) {
	students, courses, packages := nameLookups()

	lowBalance := []models.Enrollment{}

	uadmin.Filter(
		&lowBalance,
		"active = ? AND classes_remaining <= ?",
		true,
		lowBalanceThreshold,
	)

	sort.Slice(lowBalance, func(i, j int) bool {
		return lowBalance[i].ClassesRemaining <
			lowBalance[j].ClassesRemaining
	})

	rows := []renewalRow{}

	for _, e := range lowBalance {
		if len(rows) >= summaryListLimit {
			break
		}

		pkg := packages[e.PackageID]

		rows = append(rows, renewalRow{
			ID:               e.ID,
			ReferenceNumber:  e.ReferenceNumber,
			Student:          students[e.StudentID],
			Course:           courses[e.CourseID],
			Package:          pkg.Name,
			ClassesRemaining: e.ClassesRemaining,
		})
	}

	more := 0

	if len(lowBalance) > summaryListLimit {
		more = len(lowBalance) - summaryListLimit
	}

	return rows, more
}

// attendanceFollowUps returns classes that ended in the past but were never
// tagged present or absent, most recent first, capped at summaryListLimit.
// The second return value is how many more rows exist beyond that cap.
func attendanceFollowUps() ([]followUpRow, int) {
	students, courses, _ := nameLookups()

	enrollmentRows := []models.Enrollment{}
	uadmin.All(&enrollmentRows)

	enrollmentCourse := map[uint]uint{}

	for _, e := range enrollmentRows {
		enrollmentCourse[e.ID] = e.CourseID
	}

	untagged := []models.Class{}

	uadmin.Filter(
		&untagged,
		"end_time < ? AND present = ? AND absent = ?",
		time.Now(),
		false,
		false,
	)

	// Filter out malformed rows before sorting because sort.Slice below
	// dereferences StartTime.
	valid := untagged[:0]

	for _, c := range untagged {
		if c.StartTime != nil {
			valid = append(valid, c)
		}
	}

	untagged = valid

	sort.Slice(untagged, func(i, j int) bool {
		return untagged[i].StartTime.After(
			*untagged[j].StartTime,
		)
	})

	rows := []followUpRow{}

	for _, c := range untagged {
		if len(rows) >= summaryListLimit {
			break
		}

		local := c.StartTime.In(time.Local)

		rows = append(rows, followUpRow{
			ID:      c.ID,
			Student: students[c.StudentID],
			Course:  courses[enrollmentCourse[c.EnrollmentID]],
			Date:    local.Format("Jan 2, 2006"),
			DateISO: local.Format(calendarDateLayout),
			Time:    local.Format("3:04 PM"),
		})
	}

	more := 0

	if len(untagged) > summaryListLimit {
		more = len(untagged) - summaryListLimit
	}

	return rows, more
}

func sendSchedule(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	from, err := time.ParseInLocation(
		calendarDateLayout,
		query.Get("start"),
		time.Local,
	)

	if err != nil {
		dashboardFail(
			w,
			r,
			dashboardUserError("Invalid start date."),
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

	for _, s := range students {
		studentNames[s.ID] = strings.TrimSpace(
			s.FirstName + " " + s.LastName,
		)
	}

	courses := []models.Course{}
	uadmin.All(&courses)

	courseTitles := map[uint]string{}

	for _, c := range courses {
		courseTitles[c.ID] = c.Title
	}

	packages := []models.Package{}
	uadmin.All(&packages)

	packageByID := map[uint]models.Package{}

	for _, p := range packages {
		packageByID[p.ID] = p
	}

	enrollments := []models.Enrollment{}
	uadmin.All(&enrollments)

	enrollmentByID := map[uint]models.Enrollment{}
	choices := []calendarEnrollment{}

	for _, e := range enrollments {
		enrollmentByID[e.ID] = e

		if !e.Active || e.ClassesRemaining < 1 {
			continue
		}

		pkg := packageByID[e.PackageID]

		choices = append(choices, calendarEnrollment{
			ID:               e.ID,
			Student:          studentNames[e.StudentID],
			Course:           courseTitles[e.CourseID],
			Package:          pkg.Name,
			DurationMinutes:  pkg.ClassDurationInMinutes,
			ClassesRemaining: e.ClassesRemaining,
		})
	}

	rows := []models.Class{}

	uadmin.Filter(
		&rows,
		"start_time >= ? AND start_time < ?",
		from,
		to,
	)

	// Month view: per-day class counts only.
	if query.Get("view") == "month" {
		counts := map[string]int{}

		for _, c := range rows {
			if c.StartTime == nil {
				continue
			}

			counts[c.StartTime.In(time.Local).Format(calendarDateLayout)]++
		}

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status": "ok",
			"view":   "month",
			"counts": counts,
		})

		return
	}

	// Assessments + homework for the classes in range, so the detail modal can
	// render fully without a second request.
	assessments := []models.Assessment{}
	uadmin.All(&assessments)

	assessmentByClass := map[uint]models.Assessment{}

	for _, a := range assessments {
		assessmentByClass[a.ClassID] = a
	}

	homeworks := []models.Homework{}
	uadmin.All(&homeworks)

	homeworkByAssessment := map[uint]models.Homework{}

	for _, h := range homeworks {
		homeworkByAssessment[h.AssessmentID] = h
	}

	classes := []calendarClass{}

	for _, c := range rows {
		if c.StartTime == nil || c.EndTime == nil {
			continue
		}

		enr := enrollmentByID[c.EnrollmentID]
		pkg := packageByID[enr.PackageID]

		status := ""

		if c.Present {
			status = "present"
		} else if c.Absent {
			status = "absent"
		}

		var assessment *calendarAssessment

		if a, ok := assessmentByClass[c.ID]; ok {
			hw := homeworkByAssessment[a.ID]

			assessment = &calendarAssessment{
				ID:                 a.ID,
				Rating:             a.Rating,
				GrammarCorrections: a.GrammarCorrections,
				Recommendation:     a.Recommendation,
				Homework:           a.Homework,
				Remarks:            a.Remarks,
				HomeworkTitle:      hw.Title,
			}
		}

		classes = append(classes, calendarClass{
			ID:           c.ID,
			Student:      studentNames[c.StudentID],
			Course:       courseTitles[enr.CourseID],
			Package:      pkg.Name,
			Reference:    enr.ReferenceNumber,
			EnrollmentID: c.EnrollmentID,
			Start: c.StartTime.In(time.Local).Format(
				calendarJSONTimeLayout,
			),
			End: c.EndTime.In(time.Local).Format(
				calendarJSONTimeLayout,
			),
			DurationMinutes: int(
				c.EndTime.Sub(*c.StartTime).Minutes(),
			),
			Status:         status,
			CreditRefunded: c.CreditRefunded,
			Assessment:     assessment,
		})
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":      "ok",
		"view":        "day",
		"classes":     classes,
		"enrollments": choices,
	})
}

func createClass(w http.ResponseWriter, r *http.Request) {
	enrollmentID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("enrollment_id")),
		10,
		64,
	)

	if err != nil || enrollmentID == 0 {
		dashboardFail(
			w,
			r,
			dashboardUserError("Select an enrollment."),
		)
		return
	}

	start, err := time.ParseInLocation(
		calendarDateTimeLayout,
		strings.TrimSpace(r.FormValue("date"))+" "+
			strings.TrimSpace(r.FormValue("start_time")),
		time.Local,
	)

	if err != nil {
		dashboardFail(
			w,
			r,
			dashboardUserError(
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
		switch {
		case errors.Is(err, models.ErrClassEnrollmentRequired),
			errors.Is(err, models.ErrClassEnrollmentNotFound):
			dashboardFail(
				w,
				r,
				dashboardUserError("Enrollment not found."),
			)

		case errors.Is(err, models.ErrClassEnrollmentInactive):
			dashboardFail(
				w,
				r,
				dashboardUserError(
					"This enrollment is not active.",
				),
			)

		case errors.Is(err, models.ErrClassNoCreditsRemaining):
			dashboardFail(
				w,
				r,
				dashboardUserError(
					"This enrollment has no classes remaining.",
				),
			)

		case errors.Is(err, models.ErrClassDateInPast):
			dashboardFail(
				w,
				r,
				dashboardUserError(
					"The class date cannot be in the past.",
				),
			)

		case errors.Is(err, models.ErrClassStartTimeRequired),
			errors.Is(err, models.ErrClassStartTimeInPast):
			dashboardFail(
				w,
				r,
				dashboardUserError(
					"Enter a valid future start time.",
				),
			)

		case errors.Is(err, models.ErrPackageInvalidClassDuration):
			dashboardFail(
				w,
				r,
				dashboardUserError(
					"The package has no class duration set.",
				),
			)

		case errors.Is(err, models.ErrClassScheduleConflict):
			dashboardFail(
				w,
				r,
				dashboardUserError(
					"Another class is already scheduled during that time.",
				),
			)

		default:
			dashboardFail(w, r, err)
		}

		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":            "ok",
		"class_id":          class.ID,
		"classes_remaining": class.Enrollment.ClassesRemaining,
	})
}

// setAttendance tags a class present or absent. A class can only be tagged once.
//
// Attendance state and an optional credit refund are changed through the Class
// domain method so the operation is transactional and protected from generic
// persistence bypasses.
func setAttendance(w http.ResponseWriter, r *http.Request) {
	classID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("class_id")),
		10,
		64,
	)

	if err != nil || classID == 0 {
		dashboardFail(
			w,
			r,
			dashboardUserError("Invalid class."),
		)
		return
	}

	status := strings.TrimSpace(r.FormValue("status"))

	if status != "present" && status != "absent" {
		dashboardFail(
			w,
			r,
			dashboardUserError("Choose Present or Absent."),
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
		switch {
		case errors.Is(err, models.ErrClassNotFound):
			dashboardFail(
				w,
				r,
				dashboardUserError("Class not found."),
			)

		case errors.Is(err, models.ErrClassAttendanceRecorded):
			dashboardFail(
				w,
				r,
				dashboardUserError(
					"This class has already been tagged.",
				),
			)

		case errors.Is(err, models.ErrClassEnrollmentNotFound):
			dashboardFail(
				w,
				r,
				dashboardUserError(
					"The enrollment for this class could not be found.",
				),
			)

		case errors.Is(err, models.ErrClassCreditsAlreadyFull):
			dashboardFail(
				w,
				r,
				dashboardUserError(
					"The enrollment already has all of its classes available.",
				),
			)

		default:
			dashboardFail(w, r, err)
		}

		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":          "ok",
		"class_status":    status,
		"credit_refunded": class.CreditRefunded,
	})
}

// saveFeedback creates or updates the Assessment (and optional Homework) for a
// class that has been tagged present. File upload is not wired up yet;
// HomeworkFile stays empty.
func saveFeedback(w http.ResponseWriter, r *http.Request) {
	classID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("class_id")),
		10,
		64,
	)

	if err != nil || classID == 0 {
		dashboardFail(
			w,
			r,
			dashboardUserError("Invalid class."),
		)
		return
	}

	rating, err := strconv.ParseFloat(
		strings.TrimSpace(r.FormValue("rating")),
		64,
	)

	if err != nil || rating < 0 || rating > 10 {
		dashboardFail(
			w,
			r,
			dashboardUserError(
				"Enter a rating between 0 and 10.",
			),
		)
		return
	}

	db := uadmin.GetDB()

	var assessmentID uint

	err = db.Transaction(func(tx *gorm.DB) error {
		var class models.Class

		if err := tx.First(&class, uint(classID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dashboardUserError("Class not found.")
			}

			return err
		}

		if !class.Present {
			return dashboardUserError(
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

		// homework.HomeworkFile is set later, once the bucket upload
		// is wired up.
		if err := tx.Save(&homework).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		dashboardFail(w, r, err)
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":        "ok",
		"assessment_id": assessmentID,
	})
}

func dashboardFail(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	message := "Something went wrong while saving the class."

	var userErr dashboardUserError

	if errors.As(err, &userErr) {
		message = userErr.Error()
	} else {
		uadmin.Trail(
			uadmin.ERROR,
			"DashboardHandler: %v",
			err,
		)
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}
