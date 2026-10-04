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
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

var createEnrollmentInvoice = func(invoice *models.Invoice) error {
	return invoice.Create()
}

// Largest contract file accepted (10 MB).
const maxContractSize = 10 << 20

// Allowed contract file types, detected from the file's bytes (not its name).
var contractTypes = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
}

// enrollmentUserError is an error whose text is safe to show in the UI. Any other
// error is logged and replaced with a generic message.
type enrollmentUserError string

func (e enrollmentUserError) Error() string { return string(e) }

// EnrollmentHandler serves the New Enrollment page (GET) and processes its form (POST).
//
// On POST, student_type "new" creates the student first (the same way StudentHandler
// does) and then the enrollment; "existing" only creates the enrollment.
func EnrollmentHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}

	if r.Method == http.MethodPost {
		createEnrollment(w, r)
		return context
	}

	students := []models.Student{}
	uadmin.All(&students)

	courses := []models.Course{}
	uadmin.Filter(&courses, "active = ?", true)

	packages := []models.Package{}
	uadmin.Filter(&packages,
		"active = ? AND total_classes > ? AND valid_from <= ? AND valid_until >= ?",
		true,
		0,
		time.Now(),
		time.Now(),
	)

	enrollments := []models.Enrollment{}
	uadmin.All(&enrollments)
	for i := range enrollments {
		uadmin.Preload(&enrollments[i])
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

	var studentCreds models.StudentCredentials
	var enrollment models.Enrollment
	var invoice models.Invoice

	db := uadmin.GetDB()

	err = db.Transaction(func(tx *gorm.DB) error {
		if isNewStudent {
			var err error

			studentCreds, err = student.CreateWithTx(tx)
			if err != nil {
				return err
			}
		}

		enrollment = models.Enrollment{
			StudentID:        student.ID,
			CourseID:         course.ID,
			PackageID:        pkg.ID,
			TotalClasses:     total,
			ClassesRemaining: total,
			Contract:         contract,
			Active:           true,
		}

		if err := enrollment.CreateWithTx(tx); err != nil {
			return err
		}

		now := time.Now()

		invoice = models.Invoice{
			StudentID:    enrollment.StudentID,
			EnrollmentID: enrollment.ID,
			InvoiceDate:  now,
			DueDate:      now.AddDate(0, 0, 14),
			Amount:       pkg.Price,
			Paid:         false,
		}

		if err := createEnrollmentInvoiceWithTx(tx, &invoice); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		deleteContract(contract)

		uadmin.Trail(
			uadmin.ERROR,
			"EnrollmentHandler: failed to create enrollment transaction: %v",
			err,
		)

		message := "The enrollment could not be saved."

		if isNewStudent {
			message = "The student, enrollment, and invoice could not be created. Please try again."
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

// fillNewStudent validates the "Enroll New Student" fields and copies them onto student.
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

// enrollmentFormID reads a required record ID from the form.
func enrollmentFormID(r *http.Request, field, message string) (uint, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(r.FormValue(field)), 10, 64)
	if err != nil || id == 0 {
		return 0, enrollmentUserError(message)
	}
	return uint(id), nil
}

// uploadContract checks the uploaded "Contract" file (if any), sends it to the
// bucket and returns the stored path to keep in Enrollment.Contract. It returns ""
// when no file was sent.
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

	// Work out the type from the file's first bytes, never from the client's filename.
	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	ext, ok := contractTypes[http.DetectContentType(head[:n])]
	if !ok {
		return "", enrollmentUserError("The contract must be a PDF, JPG, PNG or WebP file.")
	}
	// Rewind so the upload starts from the first byte, not byte 512.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	// A unique name, so two contracts can never overwrite each other in the bucket.
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	return uploadToFilebase(file, "contract-"+hex.EncodeToString(random)+ext)
}

// deleteContract removes a contract from the bucket. Failures are only logged,
// because a leftover file isn't worth failing the request over.
func deleteContract(stored string) {
	if stored == "" {
		return
	}
	if err := deleteFromFilebase(filepath.Base(stored)); err != nil {
		uadmin.Trail(uadmin.ERROR, "Failed to delete contract from Filebase: %v", err)
	}
}

// enrollmentFail sends the error to the browser in the shape enrollment.js expects.
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

func createEnrollmentInvoiceWithTx(tx *gorm.DB, invoice *models.Invoice) error {
	return invoice.CreateWithTx(tx)
}
