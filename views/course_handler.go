package views

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
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

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "Course title is required.", http.StatusBadRequest)
			return context
		}

		levelID, err := strconv.ParseUint(
			strings.TrimSpace(r.FormValue("levelID")),
			10,
			64,
		)
		if err != nil || levelID == 0 {
			http.Error(w, "A valid level is required.", http.StatusBadRequest)
			return context
		}

		var level models.Level
		if err := uadmin.Get(&level, "id = ?", uint(levelID)); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "Level not found.", http.StatusBadRequest)
				return context
			}

			http.Error(w, "Failed to load level.", http.StatusInternalServerError)
			return context
		}

		activeStr := strings.TrimSpace(r.FormValue("active"))
		active := false

		if activeStr != "" {
			active, err = strconv.ParseBool(activeStr)
			if err != nil {
				http.Error(w, "Invalid active value.", http.StatusBadRequest)
				return context
			}
		}

		course := models.Course{
			Title:       title,
			Description: r.FormValue("description"),
			LevelID:     uint(levelID),
			Active:      active,
		}

		if err := uadmin.Save(&course); err != nil {
			http.Error(
				w,
				"Failed to save course.",
				http.StatusInternalServerError,
			)
			return context
		}

		uadmin.ReturnJSON(w, r, map[string]any{
			"status":    "ok",
			"course_id": course.ID,
		})
		return nil
	}

	if r.Method == "PUT" {
		_ = r.ParseMultipartForm(32 << 20)
		_ = r.ParseForm()

		idStr := strings.TrimSpace(r.URL.Query().Get("id"))
		if idStr == "" {
			idStr = strings.TrimSpace(r.FormValue("id"))
		}

		courseID, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || courseID == 0 {
			http.Error(w, "A valid course ID is required.", http.StatusBadRequest)
			return context
		}

		var course models.Course

		if err := uadmin.Get(
			&course,
			"id = ?",
			uint(courseID),
		); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "Course not found.", http.StatusNotFound)
				return context
			}

			http.Error(
				w,
				"Failed to load course.",
				http.StatusInternalServerError,
			)
			return context
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "Course title is required.", http.StatusBadRequest)
			return context
		}

		levelID, err := strconv.ParseUint(
			strings.TrimSpace(r.FormValue("levelID")),
			10,
			64,
		)
		if err != nil || levelID == 0 {
			http.Error(w, "A valid level is required.", http.StatusBadRequest)
			return context
		}

		var level models.Level
		if err := uadmin.Get(&level, "id = ?", uint(levelID)); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "Level not found.", http.StatusBadRequest)
				return context
			}

			http.Error(w, "Failed to load level.", http.StatusInternalServerError)
			return context
		}

		activeStr := strings.TrimSpace(r.FormValue("active"))
		active, err := strconv.ParseBool(activeStr)
		if err != nil {
			http.Error(w, "Invalid active value.", http.StatusBadRequest)
			return context
		}

		course.Title = title
		course.Description = r.FormValue("description")
		course.LevelID = uint(levelID)
		course.Active = active

		if err := uadmin.Save(&course); err != nil {
			http.Error(
				w,
				"Failed to save course.",
				http.StatusInternalServerError,
			)
			return context
		}

		uadmin.ReturnJSON(w, r, map[string]any{
			"status":    "ok",
			"course_id": course.ID,
		})
		return nil
	}

	return context
}
