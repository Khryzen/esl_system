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

	if r.Method == "POST" {
		student := models.Student{}
		student.FirstName = r.FormValue("firstName")
		student.LastName = r.FormValue("lastName")
		student.Email = r.FormValue("email")
		student.WeChatID = r.FormValue("wechatId")

		studentCreds := student.Save()
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"creds":  studentCreds,
			"status": "ok",
		})
	}
	return context
}
