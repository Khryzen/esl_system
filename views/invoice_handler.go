package views

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

// Date layout used by the Add Invoice form's <input type="date">.
const invoiceDateLayout = "2006-01-02"

// invoiceUserError is a problem with what the user submitted; its text is safe to
// show in the UI. Any other error is logged and replaced with a generic message.
type invoiceUserError string

func (e invoiceUserError) Error() string { return string(e) }

// invoiceRow is one row in the invoices table.
type invoiceRow struct {
	ID            uint
	InvoiceNumber string
	Student       string
	Course        string
	InvoiceDate   string
	DueDate       string
	Amount        float64
	TotalPaid     float64
	Balance       float64
	PaymentStatus string
	Paid          bool
	Overdue       bool
}

type invoicePaymentHistoryItem struct {
	ID              uint      `json:"id"`
	Amount          float64   `json:"amount"`
	PaymentDate     time.Time `json:"payment_date"`
	PaymentMethod   string    `json:"payment_method"`
	ReferenceNumber string    `json:"reference_number"`
	Notes           string    `json:"notes"`
}
type invoiceDetailsResponse struct {
	ID            uint       `json:"id"`
	InvoiceNumber string     `json:"invoice_number"`
	InvoiceDate   time.Time  `json:"invoice_date"`
	DueDate       time.Time  `json:"due_date"`
	Amount        float64    `json:"amount"`
	Paid          bool       `json:"paid"`
	TransactionID string     `json:"transaction_id"`
	PaidDate      *time.Time `json:"paid_date"`
	StudentID     uint       `json:"student_id"`
	StudentName   string     `json:"student_name"`
	EnrollmentID  uint       `json:"enrollment_id"`
	EnrollmentRef string     `json:"enrollment_reference"`
	CourseID      uint       `json:"course_id"`
	CourseName    string     `json:"course_name"`
	PackageID     uint       `json:"package_id"`
	PackageName   string     `json:"package_name"`
	TotalPaid     float64    `json:"total_paid"`
	Balance       float64    `json:"balance"`
	PaymentStatus string     `json:"payment_status"`
}

// invoiceEnrollmentOption is one choice in the Add Invoice form's enrollment dropdown.
// Price is used to prefill the Amount field client-side — the field stays editable,
// since a manual invoice (a materials fee, a makeup charge) won't always match the
// enrollment's package price exactly.
type invoiceEnrollmentOption struct {
	ID    uint
	Label string
	Price float64
}

// InvoiceHandler serves the Invoices page and its requests.
//
//	GET               render the page (Invoices, EnrollmentOptions, summary totals)
//	POST              create a manual invoice
//	PUT ?id=<id>      mark an invoice paid
func InvoiceHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}

	switch {
	case r.Method == http.MethodPost:
		createInvoice(w, r)
		return context
	case r.Method == http.MethodPut:
		markInvoicePaid(w, r)
		return context
	}

	students, courses, packages := nameLookups()

	// Every enrollment is needed to resolve course names on existing invoices (an
	// invoice can outlive the enrollment going inactive), but only active ones are
	// offered as choices for a new invoice.
	allEnrollments := []models.Enrollment{}
	uadmin.All(&allEnrollments)

	enrollmentCourse := map[uint]uint{}
	options := []invoiceEnrollmentOption{}
	for _, e := range allEnrollments {
		enrollmentCourse[e.ID] = e.CourseID
		if !e.Active {
			continue
		}
		pkg := packages[e.PackageID]
		options = append(options, invoiceEnrollmentOption{
			ID:    e.ID,
			Label: fmt.Sprintf("%s — %s (%s)", students[e.StudentID], courses[e.CourseID], pkg.Name),
			Price: pkg.Price,
		})
	}

	invoices := []models.Invoice{}
	uadmin.All(&invoices)
	sort.Slice(invoices, func(i, j int) bool {
		return invoices[i].InvoiceDate.After(invoices[j].InvoiceDate)
	})

	now := time.Now().In(time.Local)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	nextMonth := monthStart.AddDate(0, 1, 0)

	rows := []invoiceRow{}
	var outstanding, collectedThisMonth float64
	unpaidCount := 0

	for _, inv := range invoices {
		totalPaid, err := inv.TotalPaid()
		if err != nil {
			uadmin.Trail(
				uadmin.ERROR,
				"InvoiceHandler: failed to calculate total paid for invoice %d: %v",
				inv.ID,
				err,
			)
			continue
		}

		balance := inv.Amount - totalPaid
		if balance < 0 {
			balance = 0
		}

		paymentStatus := "Unpaid"
		if balance <= 0 {
			paymentStatus = "Paid"
		} else if totalPaid > 0 {
			paymentStatus = "Partially Paid"
		}

		hasBalance := balance > 0
		rows = append(rows, invoiceRow{
			ID:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			Student:       students[inv.StudentID],
			Course:        courses[enrollmentCourse[inv.EnrollmentID]],
			InvoiceDate:   inv.InvoiceDate.In(time.Local).Format("Jan 2, 2006"),
			DueDate:       inv.DueDate.In(time.Local).Format("Jan 2, 2006"),
			Amount:        inv.Amount,
			TotalPaid:     totalPaid,
			Balance:       balance,
			PaymentStatus: paymentStatus,
			Paid:          paymentStatus == "Paid",
			Overdue:       hasBalance && inv.DueDate.Before(now),
		})

		if hasBalance {
			outstanding += balance
			unpaidCount++
		}
	}

	if err := uadmin.GetDB().
		Model(&models.Payment{}).
		Where("payment_date >= ? AND payment_date < ?", monthStart, nextMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&collectedThisMonth).Error; err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoiceHandler: failed to calculate monthly collections: %v",
			err,
		)
	}

	context["Invoices"] = rows
	context["EnrollmentOptions"] = options
	context["OutstandingBalance"] = outstanding
	context["UnpaidCount"] = unpaidCount
	context["CollectedThisMonth"] = collectedThisMonth

	return context
}

// createInvoice handles a manually created invoice (the Add Invoice form). An
// auto-created invoice — one per new enrollment — is instead created directly in
// EnrollmentHandler, right after the enrollment itself is saved.
func createInvoice(w http.ResponseWriter, r *http.Request) {
	enrollmentID, err := strconv.ParseUint(strings.TrimSpace(r.FormValue("enrollment_id")), 10, 64)
	if err != nil || enrollmentID == 0 {
		invoiceFail(w, r, invoiceUserError("Select an enrollment."))
		return
	}

	enrollment := models.Enrollment{}
	if err := uadmin.Get(&enrollment, "id = ?", enrollmentID); err != nil {
		invoiceFail(w, r, invoiceUserError("Enrollment not found."))
		return
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("amount")), 64)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		invoiceFail(w, r, invoiceUserError("Enter an amount greater than 0."))
		return
	}
	amount = math.Round(amount*100) / 100

	dueDate, err := time.ParseInLocation(invoiceDateLayout, strings.TrimSpace(r.FormValue("due_date")), time.Local)
	if err != nil {
		invoiceFail(w, r, invoiceUserError("Enter a valid due date."))
		return
	}

	invoice := models.Invoice{
		StudentID:    enrollment.StudentID,
		EnrollmentID: enrollment.ID,
		InvoiceDate:  time.Now(),
		DueDate:      dueDate,
		Amount:       amount,
		Paid:         false,
	}

	if err := invoice.Create(); err != nil {
		invoiceFail(w, r, err)
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":         "ok",
		"invoice_id":     invoice.ID,
		"invoice_number": invoice.InvoiceNumber,
	})
}

func markInvoicePaid(w http.ResponseWriter, r *http.Request) {
	invoiceID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("invoice_id")),
		10,
		64,
	)
	if err != nil || invoiceID == 0 {
		invoiceFail(w, r, invoiceUserError("Invalid invoice."))
		return
	}

	transactionID := strings.TrimSpace(r.FormValue("transaction_id"))

	invoice := models.Invoice{
		Model: uadmin.Model{
			ID: uint(invoiceID),
		},
	}

	if err := invoice.MarkPaid(transactionID); err != nil {
		switch {
		case errors.Is(err, models.ErrInvoiceNotFound):
			invoiceFail(w, r, invoiceUserError("Invoice not found."))

		case errors.Is(err, models.ErrInvoiceAlreadyPaid):
			invoiceFail(w, r, invoiceUserError("Invoice is already paid."))

		default:
			invoiceFail(w, r, err)
		}

		return
	}

	uadmin.ReturnJSON(
		w, r,
		map[string]interface{}{
			"success":        true,
			"status":         "ok",
			"invoice_id":     invoice.ID,
			"transaction_id": invoice.TransactionID,
			"paid":           invoice.Paid,
			"paid_date":      invoice.PaidDate,
		},
	)
}

func InvoicePaymentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoicePaymentHandler: ParseMultipartForm failed: %v; Content-Type: %s",
			err,
			r.Header.Get("Content-Type"),
		)
		http.Error(w, "Invalid payment data", http.StatusBadRequest)
		return
	}

	uadmin.Trail(
		uadmin.INFO,
		"InvoicePaymentHandler: Content-Type=%s; Form=%v; PostForm=%v",
		r.Header.Get("Content-Type"),
		r.Form,
		r.PostForm,
	)

	invoiceID, err := strconv.ParseUint(
		strings.TrimSpace(r.FormValue("invoice_id")),
		10,
		64,
	)
	if err != nil || invoiceID == 0 {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(
		strings.TrimSpace(r.FormValue("amount")),
		64,
	)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		http.Error(w, "Invalid payment amount", http.StatusBadRequest)
		return
	}

	amount = math.Round(amount*100) / 100

	paymentMethod := strings.TrimSpace(r.FormValue("payment_method"))
	referenceNumber := strings.TrimSpace(r.FormValue("reference_number"))
	notes := strings.TrimSpace(r.FormValue("notes"))

	db := uadmin.GetDB()

	var invoice models.Invoice
	if err := db.First(&invoice, uint(invoiceID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Invoice not found", http.StatusNotFound)
			return
		}

		uadmin.Trail(
			uadmin.ERROR,
			"InvoicePaymentHandler: failed to load invoice %d: %v",
			invoiceID,
			err,
		)

		http.Error(w, "Could not load invoice", http.StatusInternalServerError)
		return
	}

	payment, err := invoice.RecordPayment(
		amount,
		paymentMethod,
		referenceNumber,
		notes,
	)

	paymentID := uint(0)
	if payment != nil {
		paymentID = payment.ID
	}

	uadmin.Trail(
		uadmin.INFO,
		"InvoicePaymentHandler: RecordPayment result: invoice_id=%d, amount=%.2f, method=%q, payment_id=%d, error=%v",
		invoiceID,
		amount,
		paymentMethod,
		paymentID,
		err,
	)

	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvoicePaymentAmountRequired),
			errors.Is(err, models.ErrPaymentAmountRequired),
			errors.Is(err, models.ErrPaymentMethodRequired),
			errors.Is(err, models.ErrPaymentMethodInvalid):
			http.Error(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, models.ErrInvoicePaymentExceedsBalance):
			invoiceFail(w, r, invoiceUserError(err.Error()))

		case errors.Is(err, models.ErrInvoiceAlreadyPaid):
			invoiceFail(w, r, invoiceUserError(err.Error()))

		default:
			uadmin.Trail(
				uadmin.ERROR,
				"InvoicePaymentHandler: failed to record payment for invoice %d: %v",
				invoiceID,
				err,
			)

			http.Error(w, "Could not record payment", http.StatusInternalServerError)
		}

		return
	}

	totalPaid, err := invoice.TotalPaid()
	if err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoicePaymentHandler: failed to calculate total paid for invoice %d: %v",
			invoiceID,
			err,
		)

		http.Error(w, "Could not load payment summary", http.StatusInternalServerError)
		return
	}

	balance, err := invoice.Balance()
	if err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoicePaymentHandler: failed to calculate balance for invoice %d: %v",
			invoiceID,
			err,
		)

		http.Error(w, "Could not load payment summary", http.StatusInternalServerError)
		return
	}

	status, err := invoice.PaymentStatus()
	if err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoicePaymentHandler: failed to calculate payment status for invoice %d: %v",
			invoiceID,
			err,
		)

		http.Error(w, "Could not load payment summary", http.StatusInternalServerError)
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status": "ok",
		"payment": map[string]interface{}{
			"id":               payment.ID,
			"amount":           payment.Amount,
			"payment_date":     payment.PaymentDate,
			"payment_method":   payment.PaymentMethod,
			"reference_number": payment.ReferenceNumber,
			"notes":            payment.Notes,
		},
		"invoice": map[string]interface{}{
			"id":         invoice.ID,
			"amount":     invoice.Amount,
			"total_paid": totalPaid,
			"balance":    balance,
			"status":     status,
			"paid":       invoice.Paid,
			"paid_date":  invoice.PaidDate,
		},
	})
}

// invoiceFail sends the error to the browser in the shape invoices.js expects.
func invoiceFail(w http.ResponseWriter, r *http.Request, err error) {
	message := "Something went wrong while saving the invoice."

	var userErr invoiceUserError
	if errors.As(err, &userErr) {
		message = userErr.Error()
	} else {
		uadmin.Trail(uadmin.ERROR, "InvoiceHandler: %v", err)
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}

func InvoiceDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	invoiceID, err := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	if err != nil || invoiceID == 0 {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}
	db := uadmin.GetDB()
	var invoice models.Invoice
	if err := db.
		Preload("Student").
		Preload("Enrollment").
		Preload("Enrollment.Course").
		Preload("Enrollment.Package").
		First(&invoice, uint(invoiceID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Invoice not found", http.StatusNotFound)
			return
		}

		uadmin.Trail(
			uadmin.ERROR,
			"InvoiceDetailsHandler: failed to load invoice %d: %v",
			invoiceID,
			err,
		)

		http.Error(w, "Could not load invoice", http.StatusInternalServerError)
		return
	}

	totalPaid, err := invoice.TotalPaid()
	if err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoiceDetailsHandler: failed to calculate total paid for invoice %d: %v",
			invoiceID,
			err,
		)
		http.Error(w, "Could not load invoice payment summary", http.StatusInternalServerError)
		return
	}

	balance, err := invoice.Balance()
	if err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoiceDetailsHandler: failed to calculate balance for invoice %d: %v",
			invoiceID,
			err,
		)
		http.Error(w, "Could not load invoice payment summary", http.StatusInternalServerError)
		return
	}

	paymentStatus, err := invoice.PaymentStatus()
	if err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoiceDetailsHandler: failed to calculate payment status for invoice %d: %v",
			invoiceID,
			err,
		)
		http.Error(w, "Could not load invoice payment summary", http.StatusInternalServerError)
		return
	}

	response := invoiceDetailsResponse{
		ID:            invoice.ID,
		InvoiceNumber: invoice.InvoiceNumber,
		InvoiceDate:   invoice.InvoiceDate,
		DueDate:       invoice.DueDate,
		Amount:        invoice.Amount,
		Paid:          invoice.Paid,
		TransactionID: invoice.TransactionID,
		PaidDate:      invoice.PaidDate,
		StudentID:     invoice.StudentID,
		StudentName:   invoice.Student.FirstName + " " + invoice.Student.LastName,
		EnrollmentID:  invoice.EnrollmentID,
		EnrollmentRef: invoice.Enrollment.ReferenceNumber,
		CourseID:      invoice.Enrollment.CourseID,
		CourseName:    invoice.Enrollment.Course.Title,
		PackageID:     invoice.Enrollment.PackageID,
		PackageName:   invoice.Enrollment.Package.Name,
		TotalPaid:     totalPaid,
		Balance:       balance,
		PaymentStatus: paymentStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func InvoicePaymentHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	invoiceID, err := strconv.ParseUint(
		r.URL.Query().Get("id"),
		10,
		64,
	)
	if err != nil || invoiceID == 0 {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	db := uadmin.GetDB()

	var invoice models.Invoice
	if err := db.First(&invoice, uint(invoiceID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Invoice not found", http.StatusNotFound)
			return
		}

		uadmin.Trail(
			uadmin.ERROR,
			"InvoicePaymentHistoryHandler: failed to load invoice %d: %v",
			invoiceID,
			err,
		)
		http.Error(w, "Could not load invoice", http.StatusInternalServerError)
		return
	}

	var payments []models.Payment
	if err := db.
		Where("invoice_id = ?", invoice.ID).
		Order("payment_date DESC, id DESC").
		Find(&payments).Error; err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoicePaymentHistoryHandler: failed to load payments for invoice %d: %v",
			invoiceID,
			err,
		)
		http.Error(w, "Could not load payment history", http.StatusInternalServerError)
		return
	}

	history := make([]invoicePaymentHistoryItem, 0, len(payments))
	for _, payment := range payments {
		history = append(history, invoicePaymentHistoryItem{
			ID:              payment.ID,
			Amount:          payment.Amount,
			PaymentDate:     payment.PaymentDate,
			PaymentMethod:   payment.PaymentMethod,
			ReferenceNumber: payment.ReferenceNumber,
			Notes:           payment.Notes,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"payments": history,
	}); err != nil {
		uadmin.Trail(
			uadmin.ERROR,
			"InvoicePaymentHistoryHandler: failed to encode response for invoice %d: %v",
			invoiceID,
			err,
		)
	}
}
