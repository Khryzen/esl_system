package views

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Khryzen/esl_system/models"
	"github.com/Khryzen/esl_system/utils"
	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

var (
	uploadToFilebase   = utils.UploadToFilebase
	deleteFromFilebase = utils.DeleteFromFilebase
	saveMaterial       = uadmin.Save
	saveCourseMaterial = uadmin.Save
)

type MaterialResponse struct {
	ID       uint `json:"ID"`
	Material struct {
		Name string `json:"Name"`
		File string `json:"File"`
	} `json:"Material"`
}

func CourseMaterialHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}

	switch r.Method {
	case "GET":
		courseIDStr := r.URL.Query().Get("course_id")
		courseID, _ := strconv.ParseUint(courseIDStr, 10, 64)

		materials := []models.CourseMaterial{}
		uadmin.Filter(&materials, "course_id = ?", courseID)

		responseList := []MaterialResponse{}

		for i := range materials {
			uadmin.Preload(&materials[i])
			s3Key := filepath.Base(materials[i].Material.File)
			presignedURL, err := utils.GetPresignedFileURL(s3Key)
			if err != nil {
				presignedURL = materials[i].Material.File
			}
			item := MaterialResponse{
				ID: materials[i].ID,
			}
			item.Material.Name = materials[i].Material.Name
			item.Material.File = presignedURL

			responseList = append(responseList, item)
		}

		uadmin.ReturnJSON(w, r, responseList)

	case "POST":
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": err.Error(),
			})
			return context
		}

		courseID, err := strconv.ParseUint(
			r.FormValue("courseID"),
			10,
			64,
		)
		if err != nil || courseID == 0 {
			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": "Invalid course ID",
			})
			return context
		}

		var course models.Course
		if err := uadmin.Get(
			&course,
			"id = ?",
			uint(courseID),
		); err != nil {
			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": "Course not found",
			})
			return context
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": "Upload error: " + err.Error(),
			})
			return context
		}
		defer file.Close()

		filePath, err := uploadToFilebase(file, header.Filename)
		if err != nil {
			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": err.Error(),
			})
			return context
		}

		material := models.Material{
			Name:   header.Filename,
			File:   filePath,
			Active: true,
		}

		if err := saveMaterial(&material); err != nil {
			if cleanupErr := deleteFromFilebase(filepath.Base(filePath)); cleanupErr != nil {
				uadmin.Trail(
					uadmin.ERROR,
					"Failed to clean up uploaded file: %v",
					cleanupErr,
				)
			}

			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": "Failed to save material",
			})
			return context
		}

		cm := models.CourseMaterial{
			CourseID:   uint(courseID),
			MaterialID: material.ID,
			Active:     true,
		}

		if err := saveCourseMaterial(&cm); err != nil {
			if cleanupErr := deleteFromFilebase(filepath.Base(filePath)); cleanupErr != nil {
				uadmin.Trail(
					uadmin.ERROR,
					"Failed to clean up uploaded file: %v",
					cleanupErr,
				)
			}

			if cleanupErr := uadmin.GetDB().Delete(&material).Error; cleanupErr != nil {
				uadmin.Trail(
					uadmin.ERROR,
					"Failed to clean up material: %v",
					cleanupErr,
				)
			}

			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": "Failed to save course material",
			})
			return context
		}

		uadmin.ReturnJSON(w, r, map[string]any{
			"status": "ok",
		})

		return context
	case "DELETE":
		idStr := strings.TrimSpace(r.URL.Query().Get("id"))

		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": "Invalid course material ID",
			})
			return context
		}

		db := uadmin.GetDB()

		var cm models.CourseMaterial

		if err := db.
			Preload("Material").
			First(&cm, uint(id)).
			Error; err != nil {
			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": "Record not found",
			})
			return context
		}

		filename := ""
		if cm.Material.ID != 0 {
			filename = filepath.Base(cm.Material.File)
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			if cm.Material.ID != 0 {
				if err := tx.Delete(&cm.Material).Error; err != nil {
					return err
				}
			}

			if err := tx.Delete(&cm).Error; err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			uadmin.ReturnJSON(w, r, map[string]any{
				"status":  "error",
				"message": "Failed to delete course material",
			})
			return context
		}

		if filename != "" {
			if err := deleteFromFilebase(filename); err != nil {
				uadmin.Trail(
					uadmin.ERROR,
					"Failed to delete file from Filebase after database deletion: %v",
					err,
				)
			}
		}

		uadmin.ReturnJSON(w, r, map[string]any{
			"status": "ok",
		})
	}

	return context
}
