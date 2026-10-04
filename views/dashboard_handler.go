package views

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

const (
	// An active enrollment at or below this many remaining classes shows up in
	// the "Renewals Needed" panel.
	lowBalanceThreshold = 2

	// Each dashboard sidebar panel shows at most this many rows.
	summaryListLimit   = 8
	calendarDateLayout = "2006-01-02"
)

type renewalRow struct {
	ID               uint
	ReferenceNumber  string
	Student          string
	Course           string
	Package          string
	ClassesRemaining int
	ScheduledClasses int
}

type followUpRow struct {
	ID      uint
	Student string
	Course  string
	Date    string
	DateISO string
	Time    string
}

type todayClassRow struct {
	ID          uint
	Student     string
	Course      string
	Package     string
	Start       string
	End         string
	Status      string
	StatusLabel string
	DateISO     string
}

type todayAttendanceSummary struct {
	Present int
	Absent  int
	Pending int
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	if r.Method == http.MethodPost {
		if r.FormValue("action") == "refresh_dashboard" {
			refreshDashboard(w, r)
			return map[string]interface{}{}
		}

		return map[string]interface{}{}
	}

	return dashboardContext()
}

func dashboardContext() map[string]interface{} {
	context := map[string]interface{}{}

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

	todayClasses := todaysClasses()
	context["TodayClasses"] = todayClasses
	context["TodayClassesTotal"] = len(todayClasses)

	attendance := todaysAttendance()
	context["TodayPresent"] = attendance.Present
	context["TodayAbsent"] = attendance.Absent
	context["TodayPending"] = attendance.Pending
	context["TodayAttendanceTotal"] =
		attendance.Present +
			attendance.Absent +
			attendance.Pending

	return context
}

func todaysClasses() []todayClassRow {
	students, courses, packages := nameLookups()

	now := time.Now()
	startOfToday := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		time.Local,
	)
	startOfTomorrow := startOfToday.AddDate(0, 0, 1)

	rows := []models.Class{}

	uadmin.Filter(
		&rows,
		"start_time >= ? AND start_time < ?",
		startOfToday,
		startOfTomorrow,
	)

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].StartTime == nil {
			return false
		}

		if rows[j].StartTime == nil {
			return true
		}

		return rows[i].StartTime.Before(
			*rows[j].StartTime,
		)
	})

	enrollments := []models.Enrollment{}
	uadmin.All(&enrollments)

	enrollmentByID := map[uint]models.Enrollment{}

	for _, e := range enrollments {
		enrollmentByID[e.ID] = e
	}

	result := []todayClassRow{}

	for _, c := range rows {
		if c.StartTime == nil || c.EndTime == nil {
			continue
		}

		enrollment := enrollmentByID[c.EnrollmentID]
		pkg := packages[enrollment.PackageID]

		status := ""
		statusLabel := "Upcoming"

		if c.Present {
			status = "present"
			statusLabel = "Present"
		} else if c.Absent {
			status = "absent"
			statusLabel = "Absent"
		} else if c.EndTime.Before(now) {
			status = "pending"
			statusLabel = "Attendance Pending"
		}

		localStart := c.StartTime.In(time.Local)
		localEnd := c.EndTime.In(time.Local)

		result = append(result, todayClassRow{
			ID:          c.ID,
			Student:     students[c.StudentID],
			Course:      courses[enrollment.CourseID],
			Package:     pkg.Name,
			Start:       localStart.Format("3:04 PM"),
			End:         localEnd.Format("3:04 PM"),
			Status:      status,
			StatusLabel: statusLabel,
			DateISO:     localStart.Format(calendarDateLayout),
		})
	}

	return result
}

func todaysAttendance() todayAttendanceSummary {
	now := time.Now()

	startOfToday := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		time.Local,
	)

	startOfTomorrow := startOfToday.AddDate(0, 0, 1)

	rows := []models.Class{}

	uadmin.Filter(
		&rows,
		"start_time >= ? AND start_time < ?",
		startOfToday,
		startOfTomorrow,
	)

	summary := todayAttendanceSummary{}

	for _, c := range rows {
		switch {
		case c.Present:
			summary.Present++

		case c.Absent:
			summary.Absent++

		default:
			summary.Pending++
		}
	}

	return summary
}

// refreshDashboard returns the dashboard summary data used by the
// server-rendered page. It allows dashboard.js to refresh the summary
// panels after other dashboard operations without reloading the page.
func refreshDashboard(w http.ResponseWriter, r *http.Request) {
	context := dashboardContext()

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":                      "ok",
		"number_of_students":          context["NumberOfStudents"],
		"number_of_courses":           context["NumberOfCourses"],
		"number_of_packages":          context["NumberOfPackages"],
		"renewals":                    context["Renewals"],
		"renewals_total":              context["RenewalsTotal"],
		"renewals_more":               context["RenewalsMore"],
		"attendance_follow_ups":       context["AttendanceFollowUps"],
		"attendance_follow_ups_total": context["AttendanceFollowUpsTotal"],
		"attendance_follow_ups_more":  context["AttendanceFollowUpsMore"],
	})
}

// nameLookups returns quick ID-to-name maps used by the dashboard summaries.
func nameLookups() (
	students map[uint]string,
	courses map[uint]string,
	packages map[uint]models.Package,
) {
	studentRows := []models.Student{}
	uadmin.All(&studentRows)

	students = map[uint]string{}

	for _, student := range studentRows {
		students[student.ID] = strings.TrimSpace(
			student.FirstName + " " + student.LastName,
		)
	}

	courseRows := []models.Course{}
	uadmin.All(&courseRows)

	courses = map[uint]string{}

	for _, course := range courseRows {
		courses[course.ID] = course.Title
	}

	packageRows := []models.Package{}
	uadmin.All(&packageRows)

	packages = map[uint]models.Package{}

	for _, pkg := range packageRows {
		packages[pkg.ID] = pkg
	}

	return
}

// renewalsNeeded recommends renewals based on both:
//
// 1. How many future classes are already scheduled.
// 2. How many unscheduled class credits remain.
//
// Because scheduling a class immediately deducts one credit, a zero-credit
// enrollment may still have several future classes already paid for.
//
// Renewal logic:
//   - More than lowBalanceThreshold future classes scheduled:
//     no renewal is needed yet.
//   - lowBalanceThreshold or fewer future classes scheduled:
//     check the remaining enrollment credits.
//   - If remaining credits are also at or below lowBalanceThreshold:
//     recommend renewal.
//
// Enrollment.Active is intentionally NOT used here because scheduling the
// final available credit sets Active=false even when future classes remain.
func renewalsNeeded() ([]renewalRow, int) {
	students, courses, packages := nameLookups()

	// Count future, non-cancelled classes for each enrollment.
	//
	// A scheduled class has already consumed its credit, so these classes
	// represent classes the student has already committed to attend.
	futureScheduled := map[uint]int{}

	now := time.Now()

	futureClasses := []models.Class{}

	uadmin.Filter(
		&futureClasses,
		"start_time > ? AND cancelled = ?",
		now,
		false,
	)

	for _, class := range futureClasses {
		if class.EnrollmentID == 0 {
			continue
		}

		futureScheduled[class.EnrollmentID]++
	}

	// Load all enrollments, not only active ones.
	//
	// An enrollment can be inactive because all credits were consumed by
	// scheduled future classes. It can still be a valid renewal candidate.
	enrollments := []models.Enrollment{}

	uadmin.All(&enrollments)

	candidates := []models.Enrollment{}

	for _, enrollment := range enrollments {
		scheduled := futureScheduled[enrollment.ID]

		// If the student already has plenty of future classes scheduled,
		// there is no immediate renewal need regardless of credit balance.
		if scheduled > lowBalanceThreshold {
			continue
		}

		// The schedule is low, so now check the remaining credit balance.
		if enrollment.ClassesRemaining > lowBalanceThreshold {
			continue
		}

		candidates = append(candidates, enrollment)
	}

	// Most urgent first:
	// 1. Fewest future scheduled classes.
	// 2. Fewest remaining credits.
	sort.Slice(candidates, func(i, j int) bool {
		iScheduled := futureScheduled[candidates[i].ID]
		jScheduled := futureScheduled[candidates[j].ID]

		if iScheduled != jScheduled {
			return iScheduled < jScheduled
		}

		return candidates[i].ClassesRemaining <
			candidates[j].ClassesRemaining
	})

	rows := []renewalRow{}

	for _, enrollment := range candidates {
		if len(rows) >= summaryListLimit {
			break
		}

		pkg := packages[enrollment.PackageID]
		scheduled := futureScheduled[enrollment.ID]

		rows = append(rows, renewalRow{
			ID:               enrollment.ID,
			ReferenceNumber:  enrollment.ReferenceNumber,
			Student:          students[enrollment.StudentID],
			Course:           courses[enrollment.CourseID],
			Package:          pkg.Name,
			ClassesRemaining: enrollment.ClassesRemaining,
			ScheduledClasses: scheduled,
		})
	}

	more := 0

	if len(candidates) > summaryListLimit {
		more = len(candidates) - summaryListLimit
	}

	return rows, more
}

// attendanceFollowUps returns classes that have ended but have not yet
// been tagged present or absent.
func attendanceFollowUps() ([]followUpRow, int) {
	students, courses, _ := nameLookups()

	enrollmentRows := []models.Enrollment{}
	uadmin.All(&enrollmentRows)

	enrollmentCourse := map[uint]uint{}

	for _, enrollment := range enrollmentRows {
		enrollmentCourse[enrollment.ID] = enrollment.CourseID
	}

	untagged := []models.Class{}

	uadmin.Filter(
		&untagged,
		"end_time < ? AND present = ? AND absent = ?",
		time.Now(),
		false,
		false,
	)

	valid := untagged[:0]

	for _, class := range untagged {
		if class.StartTime != nil {
			valid = append(valid, class)
		}
	}

	untagged = valid

	sort.Slice(untagged, func(i, j int) bool {
		return untagged[i].StartTime.After(
			*untagged[j].StartTime,
		)
	})

	rows := []followUpRow{}

	for _, class := range untagged {
		if len(rows) >= summaryListLimit {
			break
		}

		local := class.StartTime.In(time.Local)

		rows = append(rows, followUpRow{
			ID:      class.ID,
			Student: students[class.StudentID],
			Course:  courses[enrollmentCourse[class.EnrollmentID]],
			Date:    local.Format("Jan 2, 2006"),
			DateISO: local.Format("2006-01-02"),
			Time:    local.Format("3:04 PM"),
		})
	}

	more := 0

	if len(untagged) > summaryListLimit {
		more = len(untagged) - summaryListLimit
	}

	return rows, more
}
