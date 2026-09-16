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
		page = "student"
	default:
		page = "dashboard"
	}

	studentCount := models.Student{}
	courseCount := models.Course{}
	packageCount := models.Package{}
	context["NumberOfStudents"] = uadmin.Count(&studentCount, "id != 0")
	context["NumberOfCourses"] = uadmin.Count(&courseCount, "active = ?", true)
	context["NumberOfPackages"] = uadmin.Count(&packageCount, "active = ?", true)
	context["Page"] = strings.ToUpper(page)
	Render(w, r, page, context)
}
