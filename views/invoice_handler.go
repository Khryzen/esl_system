package views

import (
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
	Paid          bool
	Overdue       bool
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

	now := time.Now()
	rows := []invoiceRow{}
	var outstanding, collectedThisMonth float64
	unpaidCount := 0

	for _, inv := range invoices {
		rows = append(rows, invoiceRow{
			ID:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			Student:       students[inv.StudentID],
			Course:        courses[enrollmentCourse[inv.EnrollmentID]],
			InvoiceDate:   inv.InvoiceDate.In(time.Local).Format("Jan 2, 2006"),
			DueDate:       inv.DueDate.In(time.Local).Format("Jan 2, 2006"),
			Amount:        inv.Amount,
			Paid:          inv.Paid,
			Overdue:       !inv.Paid && inv.DueDate.Before(now),
		})

		if inv.Paid {
			if inv.PaidDate != nil {
				paidLocal := inv.PaidDate.In(time.Local)
				if paidLocal.Year() == now.Year() && paidLocal.Month() == now.Month() {
					collectedThisMonth += inv.Amount
				}
			}
		} else {
			outstanding += inv.Amount
			unpaidCount++
		}
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
	invoice.Save()
	if invoice.ID == 0 {
		invoiceFail(w, r, invoiceUserError("Could not save the invoice."))
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":         "ok",
		"invoice_id":     invoice.ID,
		"invoice_number": invoice.InvoiceNumber,
	})
}

// markInvoicePaid tags an invoice paid, capturing an optional transaction reference
// and the moment it was paid. There's no "unmark" — correcting a mistaken payment
// isn't supported yet and would need to be done directly against the database.
func markInvoicePaid(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseMultipartForm(32 << 20)

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	invoiceID, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64)
	if err != nil || invoiceID == 0 {
		invoiceFail(w, r, invoiceUserError("Invalid invoice."))
		return
	}

	invoice := models.Invoice{}
	if err := uadmin.Get(&invoice, "id = ?", invoiceID); err != nil {
		invoiceFail(w, r, invoiceUserError("Invoice not found."))
		return
	}
	if invoice.Paid {
		invoiceFail(w, r, invoiceUserError("This invoice is already marked paid."))
		return
	}

	now := time.Now()
	invoice.Paid = true
	invoice.PaidDate = &now
	invoice.TransactionID = strings.TrimSpace(r.FormValue("transaction_id"))

	if err := uadmin.Save(&invoice); err != nil {
		invoiceFail(w, r, err)
		return
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status": "ok",
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
