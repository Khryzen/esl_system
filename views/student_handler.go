package views

import (
	"net/http"
	"strconv"
	"strings"

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

	if r.Method == "PUT" {
		_ = r.ParseMultipartForm(32 << 20)

		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			idStr = r.FormValue("id")
		}

		studentID, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || studentID == 0 {
			uadmin.ReturnJSON(w, r, map[string]interface{}{
				"status":  "error",
				"message": "Invalid student ID",
			})
			return context
		}

		student := models.Student{}
		if err := uadmin.Get(&student, "id = ?", studentID); err != nil {
			uadmin.ReturnJSON(w, r, map[string]interface{}{
				"status":  "error",
				"message": "Student not found",
			})
			return context
		}

		student.FirstName = strings.TrimSpace(r.FormValue("firstName"))
		student.LastName = strings.TrimSpace(r.FormValue("lastName"))
		student.Email = strings.TrimSpace(r.FormValue("email"))
		student.WeChatID = strings.TrimSpace(r.FormValue("wechatId"))

		if err := uadmin.Save(&student); err != nil {
			uadmin.ReturnJSON(w, r, map[string]interface{}{
				"status":  "error",
				"message": "Could not save student",
			})
			return context
		}

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":     "ok",
			"student_id": student.ID,
		})
		return context
	}
	return context
}
