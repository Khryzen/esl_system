package views

import (
	"net/http"
	"strconv"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func CourseHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}

	allCourses := []models.Course{}
	uadmin.All(&allCourses)
	for i := range allCourses {
		uadmin.Preload(&allCourses[i])
	}

	context["AllCourses"] = allCourses

	levels := []models.Level{}
	uadmin.All(&levels)
	context["Level"] = levels

	if r.Method == "POST" {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			_ = r.ParseForm()
		}

		course := models.Course{}
		course.Title = r.FormValue("title")
		course.Description = r.FormValue("description")
		levelID, _ := strconv.ParseUint(r.FormValue("levelID"), 10, 64)
		course.LevelID = uint(levelID)
		activeStr := r.FormValue("active")
		if activeStr != "" {
			active, _ := strconv.ParseBool(activeStr)
			course.Active = active
		}
		uadmin.Save(&course)
		uadmin.ReturnJSON(w, r, map[string]any{
			"status":    "ok",
			"course_id": course.ID,
		})
		return nil
	}

	if r.Method == "PUT" {
		_ = r.ParseMultipartForm(32 << 20)
		_ = r.ParseForm()

		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			idStr = r.FormValue("id")
		}

		courseID, _ := strconv.ParseUint(idStr, 10, 64)

		course := models.Course{}
		uadmin.Get(&course, "id = ?", courseID)

		course.Title = r.FormValue("title")
		course.Description = r.FormValue("description")

		levelID, _ := strconv.ParseUint(r.FormValue("levelID"), 10, 64)
		course.LevelID = uint(levelID)

		active, _ := strconv.ParseBool(r.FormValue("active"))
		course.Active = active

		uadmin.Save(&course)
		uadmin.ReturnJSON(w, r, map[string]any{
			"status":    "ok",
			"course_id": course.ID,
		})
		return context
	}

	return context
}
