package views

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Khryzen/esl_system/models"
	"github.com/Khryzen/esl_system/utils"
	"github.com/uadmin/uadmin"
)

const maxContractSize = 10 << 20

var contractTypes = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
}

type enrollmentUserError string

func (e enrollmentUserError) Error() string { return string(e) }
func EnrollmentHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}

	if r.Method == http.MethodPost {
		createEnrollment(w, r)
		return context
	}

	students := []models.Student{}
	uadmin.All(&students)

	courses := []models.Course{}
	uadmin.All(&courses)

	packages := []models.Package{}
	uadmin.All(&packages)

	enrollments := []models.Enrollment{}
	uadmin.All(&enrollments)
	for i := range enrollments {
		uadmin.Preload(&enrollments[i])
		// uadmin.Preload(&enrollments[i].Student)
	}

	context["Enrollments"] = enrollments
	context["Students"] = students
	context["Courses"] = courses
	context["Packages"] = packages
	return context
}

func createEnrollment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxContractSize+(1<<20))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		enrollmentFail(w, r, enrollmentUserError("The form could not be read. The contract must be 10 MB or smaller."))
		return
	}

	isNewStudent := r.FormValue("student_type") == "new"

	student := models.Student{}
	if isNewStudent {
		if err := fillNewStudent(r, &student); err != nil {
			enrollmentFail(w, r, err)
			return
		}
	} else {
		studentID, err := enrollmentFormID(r, "StudentID", "Select a student.")
		if err != nil {
			enrollmentFail(w, r, err)
			return
		}
		if err := uadmin.Get(&student, "id = ?", studentID); err != nil {
			enrollmentFail(w, r, enrollmentUserError("Student not found."))
			return
		}
	}

	courseID, err := enrollmentFormID(r, "CourseID", "Select a course.")
	if err != nil {
		enrollmentFail(w, r, err)
		return
	}
	course := models.Course{}
	if err := uadmin.Get(&course, "id = ?", courseID); err != nil {
		enrollmentFail(w, r, enrollmentUserError("Course not found."))
		return
	}

	packageID, err := enrollmentFormID(r, "PackageID", "Select a package.")
	if err != nil {
		enrollmentFail(w, r, err)
		return
	}
	pkg := models.Package{}
	if err := uadmin.Get(&pkg, "id = ?", packageID); err != nil {
		enrollmentFail(w, r, enrollmentUserError("Package not found."))
		return
	}

	total := pkg.NumberOfClasses + pkg.NumberOfFreeClasses

	contract, err := uploadContract(r)
	if err != nil {
		enrollmentFail(w, r, err)
		return
	}

	var studentCreds interface{}
	if isNewStudent {
		studentCreds = student.Save()
		if student.ID == 0 {
			deleteContract(contract)
			uadmin.Trail(uadmin.ERROR, "EnrollmentHandler: the new student was not saved")
			enrollmentFail(w, r, enrollmentUserError("The student could not be created. Nothing was saved."))
			return
		}
	}

	enrollment := models.Enrollment{
		StudentID:        student.ID,
		CourseID:         course.ID,
		PackageID:        pkg.ID,
		TotalClasses:     total,
		ClassesRemaining: total,
		Contract:         contract,
		Active:           true,
	}
	enrollment.Save()
	if enrollment.ID == 0 {
		deleteContract(contract)
		uadmin.Trail(uadmin.ERROR, "EnrollmentHandler: the enrollment was not saved")

		message := "The enrollment could not be saved."
		if isNewStudent {
			message = "The student was created, but the enrollment could not be saved. " +
				"Reload the page, choose the student under \"Select Existing Student\" and try again."
		}
		enrollmentFail(w, r, enrollmentUserError(message))
		return
	}

	response := map[string]interface{}{
		"status":           "ok",
		"enrollment_id":    enrollment.ID,
		"reference_number": enrollment.ReferenceNumber,
		"student_id":       student.ID,
	}
	if isNewStudent {
		response["creds"] = studentCreds
	}
	uadmin.ReturnJSON(w, r, response)
}

func fillNewStudent(r *http.Request, student *models.Student) error {
	first := strings.TrimSpace(r.FormValue("NewStudentFirstName"))
	last := strings.TrimSpace(r.FormValue("NewStudentLastName"))
	weChat := strings.TrimSpace(r.FormValue("NewStudentWeChat"))
	email := strings.TrimSpace(r.FormValue("NewStudentEmail"))

	if first == "" || last == "" {
		return enrollmentUserError("First name and last name are required.")
	}
	if weChat == "" {
		return enrollmentUserError("WeChat ID is required.")
	}
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return enrollmentUserError("Enter a valid email address.")
	}

	student.FirstName = first
	student.LastName = last
	student.Email = email
	student.WeChatID = weChat
	return nil
}

func enrollmentFormID(r *http.Request, field, message string) (uint, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(r.FormValue(field)), 10, 64)
	if err != nil || id == 0 {
		return 0, enrollmentUserError(message)
	}
	return uint(id), nil
}

func uploadContract(r *http.Request) (string, error) {
	file, header, err := r.FormFile("Contract")
	if errors.Is(err, http.ErrMissingFile) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()

	if header.Size > maxContractSize {
		return "", enrollmentUserError("The contract must be 10 MB or smaller.")
	}

	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	ext, ok := contractTypes[http.DetectContentType(head[:n])]
	if !ok {
		return "", enrollmentUserError("The contract must be a PDF, JPG, PNG or WebP file.")
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	return utils.UploadToFilebase(file, "contract-"+hex.EncodeToString(random)+ext)
}

func deleteContract(stored string) {
	if stored == "" {
		return
	}
	if err := utils.DeleteFromFilebase(filepath.Base(stored)); err != nil {
		uadmin.Trail(uadmin.ERROR, "Failed to delete contract from Filebase: %v", err)
	}
}

func enrollmentFail(w http.ResponseWriter, r *http.Request, err error) {
	message := "Something went wrong while saving the enrollment."

	var userErr enrollmentUserError
	if errors.As(err, &userErr) {
		message = userErr.Error()
	} else {
		uadmin.Trail(uadmin.ERROR, "EnrollmentHandler: %v", err)
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}
