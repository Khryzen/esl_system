package views

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

const (
	calendarDateLayout     = "2006-01-02"
	calendarDateTimeLayout = "2006-01-02 15:04"
	calendarJSONTimeLayout = "2006-01-02T15:04:05"
)

type dashboardUserError string

func (e dashboardUserError) Error() string { return string(e) }

// calendarClass is one class block on the calendar.
type calendarClass struct {
	ID              uint   `json:"id"`
	Student         string `json:"student"`
	Course          string `json:"course"`
	Package         string `json:"package"`
	Reference       string `json:"reference"`
	EnrollmentID    uint   `json:"enrollment_id"`
	Start           string `json:"start"`
	End             string `json:"end"`
	DurationMinutes int    `json:"duration_minutes"`
	Present         bool   `json:"present"`
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
		createClass(w, r)
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

	return context
}

func sendSchedule(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	from, err := time.ParseInLocation(calendarDateLayout, query.Get("start"), time.Local)
	if err != nil {
		dashboardFail(w, r, dashboardUserError("Invalid start date."))
		return
	}
	days, err := strconv.Atoi(query.Get("days"))
	if err != nil || days < 1 || days > 62 {
		days = 1
	}
	to := from.AddDate(0, 0, days)

	// Lookups, so each class can show a student and a course name.
	students := []models.Student{}
	uadmin.All(&students)
	studentNames := map[uint]string{}
	for _, s := range students {
		studentNames[s.ID] = strings.TrimSpace(s.FirstName + " " + s.LastName)
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

	// Every enrollment is needed to label past classes, but only active ones that
	// still have classes left can be scheduled.
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
	uadmin.Filter(&rows, "start_time >= ? AND start_time < ?", from, to)

	// Month view: return per-day class counts only.
	if query.Get("view") == "month" {
		counts := map[string]int{}
		for _, c := range rows {
			if c.StartTime == nil {
				continue
			}
			key := c.StartTime.In(time.Local).Format(calendarDateLayout)
			counts[key]++
		}
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status": "ok",
			"view":   "month",
			"counts": counts,
		})
		return
	}

	// Day view: full class list.
	classes := []calendarClass{}
	for _, c := range rows {
		if c.StartTime == nil || c.EndTime == nil {
			continue
		}
		enr := enrollmentByID[c.EnrollmentID]
		pkg := packageByID[enr.PackageID]
		duration := int(c.EndTime.Sub(*c.StartTime).Minutes())
		classes = append(classes, calendarClass{
			ID:              c.ID,
			Student:         studentNames[c.StudentID],
			Course:          courseTitles[enr.CourseID],
			Package:         pkg.Name,
			Reference:       enr.ReferenceNumber,
			EnrollmentID:    c.EnrollmentID,
			Start:           c.StartTime.In(time.Local).Format(calendarJSONTimeLayout),
			End:             c.EndTime.In(time.Local).Format(calendarJSONTimeLayout),
			DurationMinutes: duration,
			Present:         c.Present,
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
	enrollmentID, err := strconv.ParseUint(strings.TrimSpace(r.FormValue("enrollment_id")), 10, 64)
	if err != nil || enrollmentID == 0 {
		dashboardFail(w, r, dashboardUserError("Select an enrollment."))
		return
	}

	start, err := time.ParseInLocation(calendarDateTimeLayout,
		strings.TrimSpace(r.FormValue("date"))+" "+strings.TrimSpace(r.FormValue("start_time")), time.Local)
	if err != nil {
		dashboardFail(w, r, dashboardUserError("Enter a valid date and start time."))
		return
	}

	enrollment := models.Enrollment{}
	if err := uadmin.Get(&enrollment, "id = ?", enrollmentID); err != nil {
		dashboardFail(w, r, dashboardUserError("Enrollment not found."))
		return
	}
	if !enrollment.Active {
		dashboardFail(w, r, dashboardUserError("This enrollment is not active."))
		return
	}
	if enrollment.ClassesRemaining < 1 {
		dashboardFail(w, r, dashboardUserError("This enrollment has no classes remaining."))
		return
	}

	pkg := models.Package{}
	if err := uadmin.Get(&pkg, "id = ?", enrollment.PackageID); err != nil {
		dashboardFail(w, r, dashboardUserError("The package for this enrollment could not be found."))
		return
	}
	if pkg.ClassDurationInMinutes < 1 {
		dashboardFail(w, r, dashboardUserError("The package has no class duration set."))
		return
	}

	// The length of the class always comes from the package, never from the browser.
	end := start.Add(time.Duration(pkg.ClassDurationInMinutes) * time.Minute)
	midnight := time.Date(start.Year(), start.Month(), start.Day()+1, 0, 0, 0, 0, time.Local)
	if end.After(midnight) {
		dashboardFail(w, r, dashboardUserError("This class would run past midnight. Choose an earlier start time."))
		return
	}

	// Only one class may run at any given time, system-wide.
	clashes := []models.Class{}
	uadmin.Filter(&clashes, "start_time < ? AND end_time > ?", end, start)
	if len(clashes) > 0 {
		dashboardFail(w, r, dashboardUserError("Another class is already scheduled during that time."))
		return
	}

	class := models.Class{
		ClassDate:    time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local),
		StartTime:    &start,
		EndTime:      &end,
		EnrollmentID: enrollment.ID,
		StudentID:    enrollment.StudentID,
	}
	if err := uadmin.Save(&class); err != nil {
		dashboardFail(w, r, err)
		return
	}

	// Booking a class uses up one of the enrollment's remaining classes.
	enrollment.ClassesRemaining--
	if err := uadmin.Save(&enrollment); err != nil {
		uadmin.Delete(&class) // undo, so a class never exists without being counted
		dashboardFail(w, r, err)
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":            "ok",
		"class_id":          class.ID,
		"classes_remaining": enrollment.ClassesRemaining,
	})
}

func dashboardFail(w http.ResponseWriter, r *http.Request, err error) {
	message := "Something went wrong while saving the class."

	var userErr dashboardUserError
	if errors.As(err, &userErr) {
		message = userErr.Error()
	} else {
		uadmin.Trail(uadmin.ERROR, "DashboardHandler: %v", err)
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}
