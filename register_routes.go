package main

import (
	"net/http"

	"github.com/Khryzen/esl_system/views"
	"github.com/uadmin/uadmin"
)

func registerRoutes() {
	http.HandleFunc("/login/", uadmin.Handler(views.LoginHandler))
	http.HandleFunc("/logout/", uadmin.Handler(views.LogoutHandler))
	http.HandleFunc("/enrollment/details/", uadmin.Handler(views.EnrollmentDetailsHandler))
	http.HandleFunc("/enrollment/change-course/", views.EnrollmentChangeCourseHandler)
	http.HandleFunc("/enrollment/deactivate/", views.EnrollmentDeactivateHandler)
	http.HandleFunc("/enrollment/invoice/", uadmin.Handler(views.EnrollmentInvoiceDetailsHandler))
	http.HandleFunc("/enrollment/renew/", views.EnrollmentRenewHandler)
	http.HandleFunc("/admin/invoice/details", views.InvoiceDetailsHandler)
	http.HandleFunc("/admin/invoice/payment", views.InvoicePaymentHandler)
	http.HandleFunc("/admin/invoice/payment-history", views.InvoicePaymentHistoryHandler)
	http.HandleFunc("/", uadmin.Handler(views.RootHandler))
}
