package views

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func setupDashboardFeedbackTestDB(t *testing.T) {
	t.Helper()
	uadmin.ClearDB()
	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/dashboard_feedback_test.db",
	}
	db := uadmin.GetDB()
	if err := db.AutoMigrate(
		&uadmin.User{},
		&models.Student{},
		&models.Course{},
		&models.Package{},
		&models.Enrollment{},
		&models.Class{},
		&models.Assessment{},
		&models.Homework{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})
}

func createDashboardFeedbackTestEnrollment(t *testing.T, classesRemaining int) models.Enrollment {
	t.Helper()
	student := models.Student{
		FirstName: "Test",
		LastName:  "Student",
	}
	if err := uadmin.Save(&student); err != nil {
		t.Fatalf("Save(student) error = %v", err)
	}
	course := models.Course{
		Title:  "Test Course",
		Active: true,
	}
	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("Save(course) error = %v", err)
	}
	pkg := models.Package{
		Name:                   "Test Package",
		NumberOfClasses:        10,
		NumberOfFreeClasses:    0,
		TotalClasses:           10,
		ClassDurationInMinutes: 30,
		Price:                  100,
		Active:                 true,
	}
	now := time.Now()
	later := now.Add(30 * 24 * time.Hour)
	pkg.ValidFrom = &now
	pkg.ValidUntil = &later
	if err := uadmin.Save(&pkg); err != nil {
		t.Fatalf("Save(package) error = %v", err)
	}
	enrollment := models.Enrollment{
		StudentID:        student.ID,
		CourseID:         course.ID,
		PackageID:        pkg.ID,
		TotalClasses:     10,
		ClassesRemaining: classesRemaining,
		ReferenceNumber:  "TEST-DASHBOARD-001",
		Active:           classesRemaining > 0,
	}
	if err := uadmin.Save(&enrollment); err != nil {
		t.Fatalf("Save(enrollment) error = %v", err)
	}
	return enrollment
}

func createDashboardFeedbackTestClass(t *testing.T, present bool) models.Class {
	t.Helper()
	enrollment := createDashboardFeedbackTestEnrollment(t, 9)
	now := time.Now()
	start := now.Add(30 * time.Minute)
	end := start.Add(30 * time.Minute)
	class := models.Class{
		ClassDate:    time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local),
		StartTime:    &start,
		EndTime:      &end,
		EnrollmentID: enrollment.ID,
		CourseID:     enrollment.CourseID,
		StudentID:    enrollment.StudentID,
		Present:      present,
		Absent:       false,
		Cancelled:    false,
	}
	if err := uadmin.Save(&class); err != nil {
		t.Fatalf("Save(class) error = %v", err)
	}
	return class
}

func createDashboardClass(t *testing.T, enrollment models.Enrollment, start, end time.Time, present, absent, cancelled bool) models.Class {
	t.Helper()
	class := models.Class{
		ClassDate:    time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local),
		StartTime:    &start,
		EndTime:      &end,
		EnrollmentID: enrollment.ID,
		CourseID:     enrollment.CourseID,
		StudentID:    enrollment.StudentID,
		Present:      present,
		Absent:       absent,
		Cancelled:    cancelled,
	}
	if err := uadmin.Save(&class); err != nil {
		t.Fatalf("Save(class) error = %v", err)
	}
	return class
}

func TestDashboardHandlerGET(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	context := DashboardHandler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/dashboard/", nil))
	if context == nil {
		t.Fatal("DashboardHandler returned nil context")
	}
	expectedKeys := []string{
		"NumberOfStudents",
		"NumberOfCourses",
		"NumberOfPackages",
		"Renewals",
		"RenewalsTotal",
		"RenewalsMore",
		"AttendanceFollowUps",
		"AttendanceFollowUpsTotal",
		"AttendanceFollowUpsMore",
		"TodayClasses",
		"TodayClassesTotal",
		"TodayPresent",
		"TodayAbsent",
		"TodayPending",
		"TodayAttendanceTotal",
	}
	for _, key := range expectedKeys {
		if _, ok := context[key]; !ok {
			t.Fatalf("%s missing from dashboard context", key)
		}
	}
}

func TestDashboardHandlerCountsActiveCoursesAndPackages(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	student := models.Student{FirstName: "Count", LastName: "Test"}
	if err := uadmin.Save(&student); err != nil {
		t.Fatalf("Save(student) error = %v", err)
	}
	activeCourse := models.Course{Title: "Active Course", Active: true}
	inactiveCourse := models.Course{Title: "Inactive Course", Active: false}
	if err := uadmin.Save(&activeCourse); err != nil {
		t.Fatalf("Save(activeCourse) error = %v", err)
	}
	if err := uadmin.Save(&inactiveCourse); err != nil {
		t.Fatalf("Save(inactiveCourse) error = %v", err)
	}
	activePackage := models.Package{Name: "Active Package", Active: true}
	inactivePackage := models.Package{Name: "Inactive Package", Active: false}
	if err := uadmin.Save(&activePackage); err != nil {
		t.Fatalf("Save(activePackage) error = %v", err)
	}
	if err := uadmin.Save(&inactivePackage); err != nil {
		t.Fatalf("Save(inactivePackage) error = %v", err)
	}
	context := dashboardContext()
	if got := context["NumberOfStudents"]; got != 1 {
		t.Fatalf("NumberOfStudents = %v, want 1", got)
	}
	if got := context["NumberOfCourses"]; got != 1 {
		t.Fatalf("NumberOfCourses = %v, want 1", got)
	}
	if got := context["NumberOfPackages"]; got != 1 {
		t.Fatalf("NumberOfPackages = %v, want 1", got)
	}
}

func TestDashboardHandlerTodayClassStatuses(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	enrollment := createDashboardFeedbackTestEnrollment(t, 9)
	now := time.Now()
	presentStart := now.Add(-2 * time.Hour)
	presentEnd := now.Add(-90 * time.Minute)
	absentStart := now.Add(-75 * time.Minute)
	absentEnd := now.Add(-45 * time.Minute)
	pendingStart := now.Add(-30 * time.Minute)
	pendingEnd := now.Add(-10 * time.Minute)
	upcomingStart := now.Add(30 * time.Minute)
	upcomingEnd := now.Add(60 * time.Minute)
	createDashboardClass(t, enrollment, presentStart, presentEnd, true, false, false)
	createDashboardClass(t, enrollment, absentStart, absentEnd, false, true, false)
	createDashboardClass(t, enrollment, pendingStart, pendingEnd, false, false, false)
	createDashboardClass(t, enrollment, upcomingStart, upcomingEnd, false, false, false)
	rows := todaysClasses()
	if len(rows) != 4 {
		t.Fatalf("todaysClasses length = %d, want 4", len(rows))
	}
	expected := []struct {
		status      string
		statusLabel string
	}{
		{"present", "Present"},
		{"absent", "Absent"},
		{"pending", "Attendance Pending"},
		{"", "Upcoming"},
	}
	for i, want := range expected {
		if rows[i].Status != want.status {
			t.Errorf("rows[%d].Status = %q, want %q", i, rows[i].Status, want.status)
		}
		if rows[i].StatusLabel != want.statusLabel {
			t.Errorf("rows[%d].StatusLabel = %q, want %q", i, rows[i].StatusLabel, want.statusLabel)
		}
	}
}

func TestDashboardHandlerTodayAttendance(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	enrollment := createDashboardFeedbackTestEnrollment(t, 9)
	now := time.Now()
	createDashboardClass(t, enrollment, now.Add(-3*time.Hour), now.Add(-150*time.Minute), true, false, false)
	createDashboardClass(t, enrollment, now.Add(-2*time.Hour), now.Add(-90*time.Minute), false, true, false)
	createDashboardClass(t, enrollment, now.Add(-60*time.Minute), now.Add(-30*time.Minute), false, false, false)
	createDashboardClass(t, enrollment, now.Add(30*time.Minute), now.Add(60*time.Minute), false, false, false)
	summary := todaysAttendance()
	if summary.Present != 1 {
		t.Fatalf("Present = %d, want 1", summary.Present)
	}
	if summary.Absent != 1 {
		t.Fatalf("Absent = %d, want 1", summary.Absent)
	}
	if summary.Pending != 2 {
		t.Fatalf("Pending = %d, want 2", summary.Pending)
	}
}

func TestDashboardHandlerAbsentAttendance(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	class := createDashboardFeedbackTestClass(t, false)
	start := time.Now().Add(-60 * time.Minute)
	end := time.Now().Add(-30 * time.Minute)
	class.StartTime = &start
	class.EndTime = &end
	class.ClassDate = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	if err := uadmin.Save(&class); err != nil {
		t.Fatalf("Save(class) error = %v", err)
	}
	if err := class.SetAttendance(false, false); err != nil {
		t.Fatalf("SetAttendance() error = %v", err)
	}
	context := DashboardHandler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/dashboard/", nil))
	if got := context["TodayPresent"]; got != 0 {
		t.Fatalf("TodayPresent = %v, want 0", got)
	}
	if got := context["TodayAbsent"]; got != 1 {
		t.Fatalf("TodayAbsent = %v, want 1", got)
	}
	if got := context["TodayPending"]; got != 0 {
		t.Fatalf("TodayPending = %v, want 0", got)
	}
	todayClasses := context["TodayClasses"].([]todayClassRow)
	if len(todayClasses) != 1 {
		t.Fatalf("TodayClasses length = %d, want 1", len(todayClasses))
	}
	if todayClasses[0].Status != "absent" {
		t.Fatalf("TodayClasses[0].Status = %q, want %q", todayClasses[0].Status, "absent")
	}
	if todayClasses[0].StatusLabel != "Absent" {
		t.Fatalf("TodayClasses[0].StatusLabel = %q, want %q", todayClasses[0].StatusLabel, "Absent")
	}
}

func TestDashboardHandlerRenewals(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	enrollment := createDashboardFeedbackTestEnrollment(t, 2)
	rows, more := renewalsNeeded()
	if more != 0 {
		t.Fatalf("renewalsNeeded more = %d, want 0", more)
	}
	if len(rows) != 1 {
		t.Fatalf("renewalsNeeded length = %d, want 1", len(rows))
	}
	if rows[0].ID != enrollment.ID {
		t.Fatalf("rows[0].ID = %d, want %d", rows[0].ID, enrollment.ID)
	}
	if rows[0].ClassesRemaining != 2 {
		t.Fatalf("rows[0].ClassesRemaining = %d, want 2", rows[0].ClassesRemaining)
	}
	if rows[0].ScheduledClasses != 0 {
		t.Fatalf("rows[0].ScheduledClasses = %d, want 0", rows[0].ScheduledClasses)
	}
}

func TestDashboardHandlerRenewalIgnoresActiveWhenCreditsAreConsumedByFutureClasses(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	enrollment := createDashboardFeedbackTestEnrollment(t, 0)
	now := time.Now()
	for i := 0; i < 2; i++ {
		start := now.Add(time.Duration(i+1) * 24 * time.Hour)
		end := start.Add(30 * time.Minute)
		createDashboardClass(t, enrollment, start, end, false, false, false)
	}
	rows, more := renewalsNeeded()
	if more != 0 {
		t.Fatalf("renewalsNeeded more = %d, want 0", more)
	}
	if len(rows) != 1 {
		t.Fatalf("renewalsNeeded length = %d, want 1", len(rows))
	}
	if rows[0].ID != enrollment.ID {
		t.Fatalf("rows[0].ID = %d, want %d", rows[0].ID, enrollment.ID)
	}
	if rows[0].ScheduledClasses != 2 {
		t.Fatalf("rows[0].ScheduledClasses = %d, want 2", rows[0].ScheduledClasses)
	}
}

func TestDashboardHandlerDoesNotRenewWithMoreThanTwoFutureClasses(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	enrollment := createDashboardFeedbackTestEnrollment(t, 9)
	enrollment.ClassesRemaining = 0
	enrollment.Active = false
	if err := uadmin.Save(&enrollment); err != nil {
		t.Fatalf("Save(enrollment) error = %v", err)
	}
	now := time.Now()
	for i := 0; i < 3; i++ {
		start := now.Add(time.Duration(i+1) * 24 * time.Hour)
		end := start.Add(30 * time.Minute)
		createDashboardClass(t, enrollment, start, end, false, false, false)
	}
	rows, _ := renewalsNeeded()
	for _, row := range rows {
		if row.ID == enrollment.ID {
			t.Fatalf("enrollment %d should not be listed for renewal with 3 future classes", enrollment.ID)
		}
	}
}

func TestDashboardHandlerCancelledFutureClassDoesNotCountForRenewal(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	enrollment := createDashboardFeedbackTestEnrollment(t, 0)
	now := time.Now()
	for i := 0; i < 3; i++ {
		start := now.Add(time.Duration(i+1) * 24 * time.Hour)
		end := start.Add(30 * time.Minute)
		createDashboardClass(t, enrollment, start, end, false, false, i == 2)
	}
	rows, _ := renewalsNeeded()
	if len(rows) != 1 {
		t.Fatalf("renewalsNeeded length = %d, want 1", len(rows))
	}
	if rows[0].ID != enrollment.ID {
		t.Fatalf("rows[0].ID = %d, want %d", rows[0].ID, enrollment.ID)
	}
	if rows[0].ScheduledClasses != 2 {
		t.Fatalf("ScheduledClasses = %d, want 2", rows[0].ScheduledClasses)
	}
}

func TestDashboardHandlerAttendanceFollowUps(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	enrollment := createDashboardFeedbackTestEnrollment(t, 9)
	now := time.Now()
	oldStart := now.Add(-3 * time.Hour)
	oldEnd := now.Add(-150 * time.Minute)
	newStart := now.Add(-90 * time.Minute)
	newEnd := now.Add(-60 * time.Minute)
	createDashboardClass(t, enrollment, oldStart, oldEnd, false, false, false)
	createDashboardClass(t, enrollment, newStart, newEnd, false, false, false)
	rows, more := attendanceFollowUps()
	if more != 0 {
		t.Fatalf("attendanceFollowUps more = %d, want 0", more)
	}
	if len(rows) != 2 {
		t.Fatalf("attendanceFollowUps length = %d, want 2", len(rows))
	}
	if rows[0].DateISO < rows[1].DateISO {
		t.Fatalf("attendance follow-ups are not ordered newest first")
	}
}

func TestDashboardHandlerAttendanceFollowUpsExcludeTaggedAndFutureClasses(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	enrollment := createDashboardFeedbackTestEnrollment(t, 9)
	now := time.Now()
	untaggedStart := now.Add(-2 * time.Hour)
	untaggedEnd := now.Add(-90 * time.Minute)
	presentStart := now.Add(-70 * time.Minute)
	presentEnd := now.Add(-40 * time.Minute)
	absentStart := now.Add(-30 * time.Minute)
	absentEnd := now.Add(-15 * time.Minute)
	futureStart := now.Add(30 * time.Minute)
	futureEnd := now.Add(60 * time.Minute)
	createDashboardClass(t, enrollment, untaggedStart, untaggedEnd, false, false, false)
	createDashboardClass(t, enrollment, presentStart, presentEnd, true, false, false)
	createDashboardClass(t, enrollment, absentStart, absentEnd, false, true, false)
	createDashboardClass(t, enrollment, futureStart, futureEnd, false, false, false)
	rows, _ := attendanceFollowUps()
	if len(rows) != 1 {
		t.Fatalf("attendanceFollowUps length = %d, want 1", len(rows))
	}
}

func TestDashboardHandlerAttendanceFollowUpsLimit(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	enrollment := createDashboardFeedbackTestEnrollment(t, 9)
	now := time.Now()
	for i := 0; i < 10; i++ {
		start := now.Add(-time.Duration(i+1) * time.Hour)
		end := start.Add(30 * time.Minute)
		createDashboardClass(t, enrollment, start, end, false, false, false)
	}
	rows, more := attendanceFollowUps()
	if len(rows) != summaryListLimit {
		t.Fatalf("attendanceFollowUps length = %d, want %d", len(rows), summaryListLimit)
	}
	if more != 2 {
		t.Fatalf("attendanceFollowUps more = %d, want 2", more)
	}
}

func TestDashboardHandlerRefresh(t *testing.T) {
	setupDashboardFeedbackTestDB(t)
	req := httptest.NewRequest(http.MethodPost, "/dashboard/", nil)
	req.Form = map[string][]string{
		"action": {"refresh_dashboard"},
	}
	rec := httptest.NewRecorder()
	context := DashboardHandler(rec, req)
	if context == nil {
		t.Fatal("DashboardHandler returned nil context")
	}
	if len(context) != 0 {
		t.Fatalf("refresh context length = %d, want 0", len(context))
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status code = %d, want %d", rec.Code, http.StatusOK)
	}
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("response status = %v, want ok", response["status"])
	}
	if _, ok := response["number_of_students"]; !ok {
		t.Fatal("number_of_students missing from refresh response")
	}
	if _, ok := response["number_of_courses"]; !ok {
		t.Fatal("number_of_courses missing from refresh response")
	}
	if _, ok := response["number_of_packages"]; !ok {
		t.Fatal("number_of_packages missing from refresh response")
	}
	if _, ok := response["renewals"]; !ok {
		t.Fatal("renewals missing from refresh response")
	}
	if _, ok := response["attendance_follow_ups"]; !ok {
		t.Fatal("attendance_follow_ups missing from refresh response")
	}
}
