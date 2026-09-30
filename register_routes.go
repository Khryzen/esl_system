package main

import (
	"net/http"

	"github.com/Khryzen/esl_system/views"
	"github.com/uadmin/uadmin"
)

func registerRoutes() {
	http.HandleFunc("/login/", uadmin.Handler(views.LoginHandler))
	http.HandleFunc("/logout/", uadmin.Handler(views.LogoutHandler))
	http.HandleFunc("/", uadmin.Handler(views.RootHandler))
}
