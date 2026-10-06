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
	"github.com/Khryzen/esl_system/utils"
	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

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

func (e enrollmentUserError) Error() string {
	return string(e)
}

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
	selectedStudentID := uint(0)

	if enrollmentID, err := strconv.ParseUint(
		strings.TrimSpace(r.URL.Query().Get("id")),
		10,
		64,
	); err == nil && enrollmentID > 0 {
		var enrollment models.Enrollment

		if err := uadmin.Get(&enrollment, "id = ?", uint(enrollmentID)); err == nil {
			selectedStudentID = enrollment.StudentID
		}
	}

	students := []models.Student{}
	uadmin.All(&students)

	courses := []models.Course{}
	uadmin.Filter(&courses, "active = ?", true)

	packages := []models.Package{}
	uadmin.Filter(&packages,
		"active = ? AND total_classes > ? AND valid_from <= ? AND valid_until >= ?",
		true, 0, time.Now(), time.Now(),
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
	context["SelectedStudentID"] = selectedStudentID
	return context
}

type enrollmentDetailsResponse struct {
	Status     string                      `json:"status"`
	Enrollment enrollmentDetailsEnrollment `json:"enrollment"`
	Invoice    *enrollmentDetailsInvoice   `json:"invoice"`
	Contract   *enrollmentDetailsContract  `json:"contract"`
	History    []enrollmentDetailsHistory  `json:"history"`
}

type enrollmentDetailsEnrollment struct {
	ID               uint   `json:"id"`
	ReferenceNumber  string `json:"reference_number"`
	StudentID        uint   `json:"student_id"`
	Student          string `json:"student"`
	CourseID         uint   `json:"course_id"`
	Course           string `json:"course"`
	Package          string `json:"package"`
	TotalClasses     int    `json:"total_classes"`
	ClassesRemaining int    `json:"classes_remaining"`
	Active           bool   `json:"active"`
}

type enrollmentDetailsInvoice struct {
	ID            uint       `json:"id"`
	InvoiceNumber string     `json:"invoice_number"`
	Amount        float64    `json:"amount"`
	InvoiceDate   time.Time  `json:"invoice_date"`
	DueDate       time.Time  `json:"due_date"`
	PaidDate      *time.Time `json:"paid_date"`
	Paid          bool       `json:"paid"`
	TransactionID string     `json:"transaction_id"`
}

type enrollmentDetailsContract struct {
	URL string `json:"url"`
}

type enrollmentDetailsHistory struct {
	ID               uint   `json:"id"`
	ReferenceNumber  string `json:"reference_number"`
	Course           string `json:"course"`
	Package          string `json:"package"`
	TotalClasses     int    `json:"total_classes"`
	ClassesRemaining int    `json:"classes_remaining"`
	Active           bool   `json:"active"`
}

func EnrollmentDetailsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(
		strings.TrimSpace(r.URL.Query().Get("id")),
		10,
		64,
	)

	if err != nil || id == 0 {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Invalid enrollment ID.",
		})
		return
	}

	db := uadmin.GetDB()

	var enrollment models.Enrollment

	if err := db.First(&enrollment, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uadmin.ReturnJSON(w, r, map[string]interface{}{
				"status":  "error",
				"message": "Enrollment not found.",
			})
			return
		}

		uadmin.Trail(
			uadmin.ERROR,
			"EnrollmentHandler: failed to load enrollment details: %v",
			err,
		)

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Could not load enrollment details.",
		})
		return
	}

	var student models.Student
	if err := db.First(&student, enrollment.StudentID).Error; err != nil {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Student not found.",
		})
		return
	}

	var course models.Course
	if err := db.First(&course, enrollment.CourseID).Error; err != nil {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Course not found.",
		})
		return
	}

	var pkg models.Package
	if err := db.First(&pkg, enrollment.PackageID).Error; err != nil {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Package not found.",
		})
		return
	}

	response := enrollmentDetailsResponse{
		Status: "ok",
		Enrollment: enrollmentDetailsEnrollment{
			ID:               enrollment.ID,
			ReferenceNumber:  enrollment.ReferenceNumber,
			StudentID:        enrollment.StudentID,
			Student:          strings.TrimSpace(student.FirstName + " " + student.LastName),
			CourseID:         enrollment.CourseID,
			Course:           course.Title,
			Package:          pkg.Name,
			TotalClasses:     enrollment.TotalClasses,
			ClassesRemaining: enrollment.ClassesRemaining,
			Active:           enrollment.Active,
		},
		History: []enrollmentDetailsHistory{},
	}

	if enrollment.Contract != "" {
		contractURL, err := utils.GetPresignedFileURL(filepath.Base(enrollment.Contract))
		if err != nil {
			uadmin.Trail(
				uadmin.ERROR,
				"EnrollmentHandler: failed to create contract URL: %v",
				err,
			)
		} else {
			response.Contract = &enrollmentDetailsContract{
				URL: contractURL,
			}
		}
	}

	var invoice models.Invoice

	if err := db.
		Where("enrollment_id = ?", enrollment.ID).
		Order("id DESC").
		First(&invoice).Error; err == nil {
		response.Invoice = &enrollmentDetailsInvoice{
			ID:            invoice.ID,
			InvoiceNumber: invoice.InvoiceNumber,
			Amount:        invoice.Amount,
			InvoiceDate:   invoice.InvoiceDate,
			DueDate:       invoice.DueDate,
			PaidDate:      invoice.PaidDate,
			Paid:          invoice.Paid,
			TransactionID: invoice.TransactionID,
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		uadmin.Trail(
			uadmin.ERROR,
			"EnrollmentHandler: failed to load enrollment invoice: %v",
			err,
		)
	}

	var history []models.Enrollment

	if err := db.
		Where("student_id = ? AND id <> ?", enrollment.StudentID, enrollment.ID).
		Order("id DESC").
		Find(&history).Error; err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"EnrollmentHandler: failed to load enrollment history: %v",
			err,
		)
	} else {
		for _, item := range history {
			var historyCourse models.Course
			var historyPackage models.Package

			if err := db.First(&historyCourse, item.CourseID).Error; err != nil {
				continue
			}

			if err := db.First(&historyPackage, item.PackageID).Error; err != nil {
				continue
			}

			response.History = append(
				response.History,
				enrollmentDetailsHistory{
					ID:               item.ID,
					ReferenceNumber:  item.ReferenceNumber,
					Course:           historyCourse.Title,
					Package:          historyPackage.Name,
					TotalClasses:     item.TotalClasses,
					ClassesRemaining: item.ClassesRemaining,
					Active:           item.Active,
				},
			)
		}
	}

	uadmin.ReturnJSON(w, r, response)
}

func EnrollmentInvoiceDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Method not allowed.",
		})
		return
	}

	enrollmentID, err := strconv.ParseUint(
		strings.TrimSpace(r.URL.Query().Get("enrollment_id")),
		10,
		64,
	)

	if err != nil || enrollmentID == 0 {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Invalid enrollment ID.",
		})
		return
	}

	db := uadmin.GetDB()

	var enrollment models.Enrollment

	if err := db.First(&enrollment, uint(enrollmentID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uadmin.ReturnJSON(w, r, map[string]interface{}{
				"status":  "error",
				"message": "Enrollment not found.",
			})
			return
		}

		uadmin.Trail(
			uadmin.ERROR,
			"EnrollmentInvoiceDetailsHandler: failed to load enrollment: %v",
			err,
		)

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Could not load enrollment invoice.",
		})
		return
	}

	var invoice models.Invoice

	if err := db.
		Where("enrollment_id = ?", enrollment.ID).
		Order("id DESC").
		First(&invoice).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			uadmin.ReturnJSON(w, r, map[string]interface{}{
				"status":  "error",
				"message": "No invoice is associated with this enrollment.",
			})
			return
		}

		uadmin.Trail(
			uadmin.ERROR,
			"EnrollmentInvoiceDetailsHandler: failed to load invoice: %v",
			err,
		)

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Could not load enrollment invoice.",
		})
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status": "ok",
		"invoice": enrollmentDetailsInvoice{
			ID:            invoice.ID,
			InvoiceNumber: invoice.InvoiceNumber,
			Amount:        invoice.Amount,
			InvoiceDate:   invoice.InvoiceDate,
			DueDate:       invoice.DueDate,
			PaidDate:      invoice.PaidDate,
			Paid:          invoice.Paid,
			TransactionID: invoice.TransactionID,
		},
	})
}

func EnrollmentChangeCourseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Method not allowed.",
		})
		return
	}

	enrollmentID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("enrollment_id")),
		10,
		64,
	)

	if err != nil || enrollmentID == 0 {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Invalid enrollment ID.",
		})
		return
	}

	courseID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("course_id")),
		10,
		64,
	)

	if err != nil || courseID == 0 {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Select a course.",
		})
		return
	}

	enrollment := models.Enrollment{
		Model: uadmin.Model{
			ID: uint(enrollmentID),
		},
	}

	if err := enrollment.ChangeCourse(uint(courseID)); err != nil {
		message := enrollmentChangeCourseError(err)

		if message == "" {
			uadmin.Trail(
				uadmin.ERROR,
				"EnrollmentChangeCourseHandler: failed to change course: %v",
				err,
			)

			message = "The enrollment course could not be changed. Please try again."
		}

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": message,
		})
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":        "ok",
		"enrollment_id": enrollment.ID,
		"course_id":     enrollment.CourseID,
	})
}

func enrollmentChangeCourseError(err error) string {
	switch {
	case errors.Is(err, models.ErrEnrollmentChangeCourseRequired):
		return "Select a course."

	case errors.Is(err, models.ErrEnrollmentNotFound):
		return "Enrollment not found."

	case errors.Is(err, models.ErrEnrollmentChangeCourseNoCredits):
		return "This enrollment has no classes remaining."

	case errors.Is(err, models.ErrEnrollmentChangeCourseSame):
		return "The enrollment is already assigned to this course."

	case errors.Is(err, models.ErrEnrollmentCourseNotFound):
		return "The selected course could not be found."

	case errors.Is(err, models.ErrEnrollmentChangeCourseInactive):
		return "The selected course is not available."

	case errors.Is(err, models.ErrEnrollmentAlreadyExists):
		return "The student already has an active enrollment for this course."

	default:
		return ""
	}
}

func createEnrollment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxContractSize+(1<<20))

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		enrollmentFail(
			w,
			r,
			enrollmentUserError("The form could not be read. The contract must be 10 MB or smaller."),
		)
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

		enrollmentFail(w, r, enrollmentTransactionError(err, isNewStudent))
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

	if err != nil &&
		!errors.Is(err, io.EOF) &&
		!errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}

	ext, ok := contractTypes[http.DetectContentType(head[:n])]

	if !ok {
		return "", enrollmentUserError(
			"The contract must be a PDF, JPG, PNG or WebP file.",
		)
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

	return uploadToFilebase(
		file,
		"contract-"+hex.EncodeToString(random)+ext,
	)
}

// deleteContract removes a contract from the bucket. Failures are only logged,
// because a leftover file isn't worth failing the request over.
func deleteContract(stored string) {
	if stored == "" {
		return
	}

	if err := deleteFromFilebase(filepath.Base(stored)); err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"Failed to delete contract from Filebase: %v",
			err,
		)
	}
}

func enrollmentTransactionError(err error, isNewStudent bool) error {
	switch {
	case errors.Is(err, models.ErrEnrollmentAlreadyExists):
		return enrollmentUserError("The student already has an active enrollment for this course.")
	case errors.Is(err, models.ErrEnrollmentCourseInactive):
		if isNewStudent {
			return enrollmentUserError("The student, enrollment, and invoice could not be created. Please try again.")
		}
		return enrollmentUserError("The selected course is not available for enrollment.")
	case errors.Is(err, models.ErrEnrollmentPackageInactive):
		if isNewStudent {
			return enrollmentUserError("The student, enrollment, and invoice could not be created. Please try again.")
		}
		return enrollmentUserError("The selected package is not available for enrollment.")
	case errors.Is(err, models.ErrEnrollmentPackageExpired):
		if isNewStudent {
			return enrollmentUserError("The student, enrollment, and invoice could not be created. Please try again.")
		}
		return enrollmentUserError("The selected package is outside its validity period.")
	case errors.Is(err, models.ErrEnrollmentPackageNoClasses):
		if isNewStudent {
			return enrollmentUserError("The student, enrollment, and invoice could not be created. Please try again.")
		}
		return enrollmentUserError("The selected package has no available classes.")
	case errors.Is(err, models.ErrEnrollmentStudentRequired):
		return enrollmentUserError("Select a student.")
	case errors.Is(err, models.ErrEnrollmentCourseRequired):
		return enrollmentUserError("Select a course.")
	case errors.Is(err, models.ErrEnrollmentPackageRequired):
		return enrollmentUserError("Select a package.")
	case errors.Is(err, models.ErrEnrollmentStudentNotFound):
		return enrollmentUserError("The selected student could not be found.")
	case errors.Is(err, models.ErrEnrollmentCourseNotFound):
		return enrollmentUserError("The selected course could not be found.")
	case errors.Is(err, models.ErrEnrollmentPackageNotFound):
		return enrollmentUserError("The selected package could not be found.")
	default:
		if isNewStudent {
			return enrollmentUserError("The student, enrollment, and invoice could not be created. Please try again.")
		}
		return enrollmentUserError("The enrollment could not be saved.")
	}
}

func EnrollmentDeactivateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Method not allowed.",
		})
		return
	}

	enrollmentID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("enrollment_id")),
		10,
		64,
	)

	if err != nil || enrollmentID == 0 {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Invalid enrollment ID.",
		})
		return
	}

	enrollment := models.Enrollment{
		Model: uadmin.Model{
			ID: uint(enrollmentID),
		},
	}

	if err := enrollment.Deactivate(); err != nil {
		message := enrollmentDeactivateError(err)

		if message == "" {
			uadmin.Trail(
				uadmin.ERROR,
				"EnrollmentDeactivateHandler: failed to deactivate enrollment: %v",
				err,
			)

			message = "The enrollment could not be deactivated. Please try again."
		}

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": message,
		})
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":        "ok",
		"enrollment_id": enrollment.ID,
		"active":        enrollment.Active,
	})
}

func EnrollmentRenewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Method not allowed.",
		})
		return
	}

	enrollmentID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("enrollment_id")),
		10,
		64,
	)

	if err != nil || enrollmentID == 0 {
		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "Invalid enrollment ID.",
		})
		return
	}

	courseID, err := enrollmentFormID(
		r,
		"course_id",
		"Select a course.",
	)

	if err != nil {
		enrollmentFail(w, r, err)
		return
	}

	packageID, err := enrollmentFormID(
		r,
		"package_id",
		"Select a package.",
	)

	if err != nil {
		enrollmentFail(w, r, err)
		return
	}

	db := uadmin.GetDB()

	var original models.Enrollment

	if err := db.First(&original, uint(enrollmentID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uadmin.ReturnJSON(w, r, map[string]interface{}{
				"status":  "error",
				"message": "The enrollment to renew could not be found.",
			})
			return
		}

		uadmin.Trail(
			uadmin.ERROR,
			"EnrollmentRenewHandler: failed to load enrollment: %v",
			err,
		)

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": "The enrollment could not be renewed. Please try again.",
		})
		return
	}

	var pkg models.Package

	if err := db.First(&pkg, packageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			enrollmentFail(
				w,
				r,
				enrollmentUserError("The selected package could not be found."),
			)
			return
		}

		uadmin.Trail(
			uadmin.ERROR,
			"EnrollmentRenewHandler: failed to load package: %v",
			err,
		)

		enrollmentFail(
			w,
			r,
			enrollmentUserError("The selected package could not be loaded."),
		)
		return
	}

	contract, err := uploadContract(r)
	if err != nil {
		enrollmentFail(w, r, err)
		return
	}

	var renewed *models.Enrollment
	var invoice models.Invoice

	err = db.Transaction(func(tx *gorm.DB) error {
		var err error

		renewed, err = original.RenewWithTx(
			tx,
			courseID,
			packageID,
			contract,
		)

		if err != nil {
			return err
		}

		now := time.Now()

		invoice = models.Invoice{
			StudentID:    renewed.StudentID,
			EnrollmentID: renewed.ID,
			InvoiceDate:  now,
			DueDate:      now.AddDate(0, 0, 14),
			Amount:       pkg.Price,
			Paid:         false,
		}

		if err := createRenewalInvoiceWithTx(tx, &invoice); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		deleteContract(contract)

		message := enrollmentRenewalError(err)

		if message == "" {
			uadmin.Trail(
				uadmin.ERROR,
				"EnrollmentRenewHandler: failed to renew enrollment: %v",
				err,
			)

			message = "The enrollment could not be renewed. Please try again."
		}

		uadmin.ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"message": message,
		})
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":           "ok",
		"enrollment_id":    renewed.ID,
		"reference_number": renewed.ReferenceNumber,
		"invoice_id":       invoice.ID,
	})
}

func enrollmentRenewalError(err error) string {
	switch {
	case errors.Is(err, models.ErrEnrollmentRenewalNotFound):
		return "The enrollment to renew could not be found."

	case errors.Is(err, models.ErrEnrollmentRenewalInactive):
		return "This enrollment is inactive and cannot be renewed."

	case errors.Is(err, models.ErrEnrollmentCourseRequired):
		return "Select a course."

	case errors.Is(err, models.ErrEnrollmentPackageRequired):
		return "Select a package."

	case errors.Is(err, models.ErrEnrollmentCourseNotFound):
		return "The selected course could not be found."

	case errors.Is(err, models.ErrEnrollmentPackageNotFound):
		return "The selected package could not be found."

	case errors.Is(err, models.ErrEnrollmentCourseInactive):
		return "The selected course is not available for enrollment."

	case errors.Is(err, models.ErrEnrollmentPackageInactive):
		return "The selected package is not available for enrollment."

	case errors.Is(err, models.ErrEnrollmentPackageExpired):
		return "The selected package is outside its validity period."

	case errors.Is(err, models.ErrEnrollmentPackageNoClasses):
		return "The selected package has no available classes."

	case errors.Is(err, models.ErrEnrollmentAlreadyExists):
		return "The student already has an active enrollment for this course."

	default:
		return ""
	}
}

func enrollmentDeactivateError(err error) string {
	switch {
	case errors.Is(err, models.ErrEnrollmentNotFound):
		return "Enrollment not found."
	case errors.Is(err, models.ErrEnrollmentDeactivateAlreadyInactive):
		return "This enrollment is already inactive."
	default:
		return ""
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

var createEnrollmentInvoiceWithTx = func(
	tx *gorm.DB,
	invoice *models.Invoice,
) error {
	return invoice.CreateWithTx(tx)
}

var createRenewalInvoiceWithTx = func(
	tx *gorm.DB,
	invoice *models.Invoice,
) error {
	return invoice.CreateWithTx(tx)
}
