package views

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/Khryzen/esl_system/models"
	"github.com/Khryzen/esl_system/utils"
	"github.com/uadmin/uadmin"
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
			uadmin.ReturnJSON(w, r, map[string]any{"status": "error", "message": err.Error()})
			return context
		}

		courseID, _ := strconv.ParseUint(r.FormValue("courseID"), 10, 64)
		file, header, err := r.FormFile("file")
		if err != nil {
			uadmin.ReturnJSON(w, r, map[string]any{"status": "error", "message": "Upload error: " + err.Error()})
			return context
		}
		defer file.Close()

		filePath, err := utils.UploadToFilebase(file, header.Filename)
		if err != nil {
			uadmin.ReturnJSON(w, r, map[string]any{"status": "error", "message": err.Error()})
			return context
		}
		material := models.Material{
			Name: header.Filename,
			File: filePath,
		}
		uadmin.Save(&material)

		cm := models.CourseMaterial{
			CourseID:   uint(courseID),
			MaterialID: material.ID,
		}
		uadmin.Save(&cm)

		uadmin.ReturnJSON(w, r, map[string]any{"status": "ok"})

	case "DELETE":
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.ParseUint(idStr, 10, 64)

		cm := models.CourseMaterial{}
		if uadmin.Get(&cm, "id = ?", id) != nil {
			uadmin.Preload(&cm)
			if cm.Material.ID != 0 {
				filename := filepath.Base(cm.Material.File)

				if err := utils.DeleteFromFilebase(filename); err != nil {
					uadmin.Trail(uadmin.ERROR, "Failed to delete file from Filebase: %v", err)
				}
				uadmin.Delete(&cm.Material)
			}
			uadmin.Delete(&cm)
			uadmin.ReturnJSON(w, r, map[string]any{"status": "ok"})
		} else {
			uadmin.ReturnJSON(w, r, map[string]any{"status": "error", "message": "Record not found"})
		}
	}

	return context
}
