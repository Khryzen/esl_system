package views

import (
	"net/http"
	"strings"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	session := uadmin.IsAuthenticated(r)
	if session == nil {
		http.Redirect(w, r, "/login/", http.StatusSeeOther)
		return
	}

	page := strings.TrimPrefix(r.URL.Path, "/")
	page = strings.TrimSuffix(page, "/")

	context := map[string]interface{}{}

	switch page {
	case "":
		context = DashboardHandler(w, r)
		page = "dashboard"
	case "student":
		context = StudentHandler(w, r)
		page = "student"
	case "course":
		context = CourseHandler(w, r)
		page = "course"
	case "package":
		context = PackageHandler(w, r)
		page = "package"
	case "enrollment":
		context = EnrollmentHandler(w, r)
		page = "enrollment"
	case "course-materials":
		context = CourseMaterialHandler(w, r)
		return
	default:
		page = "dashboard"
	}

	if r.Method != "GET" {
		return
	}

	if context == nil {
		context = map[string]interface{}{}
	}

	studentCount := []models.Student{}
	uadmin.Filter(&studentCount, "id > 0")

	courseCount := models.Course{}
	packageCount := models.Package{}
	username, _ := GetCookieValue(r, "username")

	user := uadmin.User{}
	uadmin.Get(&user, "username = ?", username)

	context["FirstName"] = user.FirstName
	context["LastName"] = user.LastName
	context["NumberOfStudents"] = len(studentCount)
	context["NumberOfCourses"] = uadmin.Count(&courseCount, "active = ?", true)
	context["NumberOfPackages"] = uadmin.Count(&packageCount, "active = ?", true)
	context["Page"] = strings.ToUpper(page)
	Render(w, r, page, context)
}

func GetCookieValue(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}
