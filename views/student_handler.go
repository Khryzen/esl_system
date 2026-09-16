package views

import (
	"net/http"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func StudentHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}

	// Fetch all students
	students := []models.Student{}
	uadmin.All(&students)
	context["AllStudents"] = students

	return context
}
