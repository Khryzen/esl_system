package views

import (
	"net/http"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func EnrollmentHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}
	students := []models.Student{}
	uadmin.All(&students)

	courses := []models.Course{}
	uadmin.All(&courses)

	packages := []models.Package{}
	uadmin.All(&packages)

	context["Courses"] = courses
	context["Packages"] = packages
	context["Students"] = students
	return context
}
