package views

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func setupCourseMaterialHandlerTestDB(t *testing.T) {
	t.Helper()

	uadmin.ClearDB()

	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/course_material_handler_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&models.Level{},
		&models.Course{},
		&models.Material{},
		&models.CourseMaterial{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})
}

func createCourseMaterialHandlerTestCourse(t *testing.T) models.Course {
	t.Helper()

	level := models.Level{
		Level: "Beginner",
	}

	if err := uadmin.Save(&level); err != nil {
		t.Fatalf("failed to create level: %v", err)
	}

	course := models.Course{
		Title:   "General English",
		LevelID: level.ID,
		Active:  true,
	}

	if err := uadmin.Save(&course); err != nil {
		t.Fatalf("failed to create course: %v", err)
	}

	return course
}

func TestCourseMaterialHandlerPOSTRejectsInvalidCourseID(t *testing.T) {
	setupCourseMaterialHandlerTestDB(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/course-material",
		strings.NewReader("courseID=not-a-number"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	CourseMaterialHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&models.CourseMaterial{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count course materials: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"course material count = %d, want 0",
			count,
		)
	}
}

func TestCourseMaterialHandlerPOSTRejectsMissingCourse(t *testing.T) {
	setupCourseMaterialHandlerTestDB(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/course-material",
		strings.NewReader("courseID=999999"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	CourseMaterialHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&models.CourseMaterial{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count course materials: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"course material count = %d, want 0",
			count,
		)
	}
}

func TestCourseMaterialHandlerDELETERejectsInvalidID(t *testing.T) {
	setupCourseMaterialHandlerTestDB(t)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/course-material?id=not-a-number",
		nil,
	)

	rec := httptest.NewRecorder()

	CourseMaterialHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&models.CourseMaterial{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count course materials: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"course material count = %d, want 0",
			count,
		)
	}
}

func TestCourseMaterialHandlerDELETEMissingRecord(t *testing.T) {
	setupCourseMaterialHandlerTestDB(t)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/course-material?id=999999",
		nil,
	)

	rec := httptest.NewRecorder()

	CourseMaterialHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	var count int64

	if err := uadmin.GetDB().
		Model(&models.CourseMaterial{}).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count course materials: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"course material count = %d, want 0",
			count,
		)
	}
}

func TestCourseMaterialSaveIntegrity(t *testing.T) {
	setupCourseMaterialHandlerTestDB(t)

	course1 := models.Course{
		Title:  "Course 1",
		Active: true,
	}
	if err := uadmin.Save(&course1); err != nil {
		t.Fatalf("failed to save course1: %v", err)
	}

	course2 := models.Course{
		Title:  "Course 2",
		Active: true,
	}
	if err := uadmin.Save(&course2); err != nil {
		t.Fatalf("failed to save course2: %v", err)
	}

	material1 := models.Material{
		Name:   "Material 1",
		File:   "material-1.pdf",
		Active: true,
	}
	if err := uadmin.Save(&material1); err != nil {
		t.Fatalf("failed to save material1: %v", err)
	}

	material2 := models.Material{
		Name:   "Material 2",
		File:   "material-2.pdf",
		Active: true,
	}
	if err := uadmin.Save(&material2); err != nil {
		t.Fatalf("failed to save material2: %v", err)
	}

	cm := models.CourseMaterial{
		CourseID:   course1.ID,
		MaterialID: material1.ID,
		Active:     true,
	}

	if err := uadmin.Save(&cm); err != nil {
		t.Fatalf("failed to save course material: %v", err)
	}

	cm.CourseID = course2.ID
	cm.MaterialID = material2.ID

	if err := uadmin.Save(&cm); err != nil {
		t.Fatalf("failed to update course material: %v", err)
	}

	var saved models.CourseMaterial

	if err := uadmin.Get(
		&saved,
		"id = ?",
		cm.ID,
	); err != nil {
		t.Fatalf("failed to reload course material: %v", err)
	}

	if saved.CourseID != course1.ID {
		t.Fatalf(
			"CourseID changed from %d to %d",
			course1.ID,
			saved.CourseID,
		)
	}

	if saved.MaterialID != material1.ID {
		t.Fatalf(
			"MaterialID changed from %d to %d",
			material1.ID,
			saved.MaterialID,
		)
	}
}

func newCourseMaterialUploadRequest(t *testing.T, courseID uint) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("courseID", strconv.FormatUint(uint64(courseID), 10)); err != nil {
		t.Fatalf("failed to write courseID: %v", err)
	}

	part, err := writer.CreateFormFile("file", "lesson.pdf")
	if err != nil {
		t.Fatalf("failed to create file field: %v", err)
	}

	if _, err := part.Write([]byte("test file")); err != nil {
		t.Fatalf("failed to write file contents: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/course-material",
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

func TestCourseMaterialHandlerPOSTCleansUpFileWhenMaterialSaveFails(t *testing.T) {
	setupCourseMaterialHandlerTestDB(t)

	course := createCourseMaterialHandlerTestCourse(t)

	oldUpload := uploadToFilebase
	oldDelete := deleteFromFilebase
	oldSaveMaterial := saveMaterial
	oldSaveCourseMaterial := saveCourseMaterial

	defer func() {
		uploadToFilebase = oldUpload
		deleteFromFilebase = oldDelete
		saveMaterial = oldSaveMaterial
		saveCourseMaterial = oldSaveCourseMaterial
	}()

	var deletedFilename string

	uploadToFilebase = func(file multipart.File, filename string) (string, error) {
		return "https://bucket.s3.filebase.io/lesson.pdf", nil
	}

	deleteFromFilebase = func(filename string) error {
		deletedFilename = filename
		return nil
	}

	saveMaterial = func(value interface{}) error {
		return errors.New("forced material save failure")
	}

	saveCourseMaterial = uadmin.Save

	req := newCourseMaterialUploadRequest(t, course.ID)
	rec := httptest.NewRecorder()

	CourseMaterialHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	if deletedFilename != "lesson.pdf" {
		t.Fatalf(
			"deleted filename = %q, want %q",
			deletedFilename,
			"lesson.pdf",
		)
	}

	var materialCount int64
	if err := uadmin.GetDB().
		Model(&models.Material{}).
		Count(&materialCount).Error; err != nil {
		t.Fatalf("failed to count materials: %v", err)
	}

	if materialCount != 0 {
		t.Fatalf("material count = %d, want 0", materialCount)
	}

	var courseMaterialCount int64
	if err := uadmin.GetDB().
		Model(&models.CourseMaterial{}).
		Count(&courseMaterialCount).Error; err != nil {
		t.Fatalf("failed to count course materials: %v", err)
	}

	if courseMaterialCount != 0 {
		t.Fatalf(
			"course material count = %d, want 0",
			courseMaterialCount,
		)
	}
}

func TestCourseMaterialHandlerPOSTCleansUpMaterialWhenCourseMaterialSaveFails(t *testing.T) {
	setupCourseMaterialHandlerTestDB(t)

	course := createCourseMaterialHandlerTestCourse(t)

	oldUpload := uploadToFilebase
	oldDelete := deleteFromFilebase
	oldSaveMaterial := saveMaterial
	oldSaveCourseMaterial := saveCourseMaterial

	defer func() {
		uploadToFilebase = oldUpload
		deleteFromFilebase = oldDelete
		saveMaterial = oldSaveMaterial
		saveCourseMaterial = oldSaveCourseMaterial
	}()

	var deletedFilename string

	uploadToFilebase = func(file multipart.File, filename string) (string, error) {
		return "https://bucket.s3.filebase.io/lesson.pdf", nil
	}

	deleteFromFilebase = func(filename string) error {
		deletedFilename = filename
		return nil
	}

	saveMaterial = uadmin.Save

	saveCourseMaterial = func(value interface{}) error {
		return errors.New("forced course material save failure")
	}

	req := newCourseMaterialUploadRequest(t, course.ID)
	rec := httptest.NewRecorder()

	CourseMaterialHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	if deletedFilename != "lesson.pdf" {
		t.Fatalf(
			"deleted filename = %q, want %q",
			deletedFilename,
			"lesson.pdf",
		)
	}

	var materialCount int64
	if err := uadmin.GetDB().
		Model(&models.Material{}).
		Count(&materialCount).Error; err != nil {
		t.Fatalf("failed to count materials: %v", err)
	}

	if materialCount != 0 {
		t.Fatalf("material count = %d, want 0", materialCount)
	}

	var courseMaterialCount int64
	if err := uadmin.GetDB().
		Model(&models.CourseMaterial{}).
		Count(&courseMaterialCount).Error; err != nil {
		t.Fatalf("failed to count course materials: %v", err)
	}

	if courseMaterialCount != 0 {
		t.Fatalf(
			"course material count = %d, want 0",
			courseMaterialCount,
		)
	}
}
