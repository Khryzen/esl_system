package main

import (
	"net/http"

	"github.com/Khryzen/esl_system/models"
	"github.com/Khryzen/esl_system/views"
	"github.com/uadmin/uadmin"
)

func setupApplication(config AppConfig) {
	if config.Database != nil {
		uadmin.Database = config.Database
	}

	uadmin.Register(
		models.Assessment{},
		models.Class{},
		models.Course{},
		models.CourseMaterial{},
		models.Enrollment{},
		models.Homework{},
		models.Invoice{},
		models.Level{},
		models.Material{},
		models.Package{},
		models.Student{},
		models.Teacher{},
	)

	initializeData()

	http.HandleFunc("/login/", uadmin.Handler(views.LoginHandler))
	http.HandleFunc("/logout/", uadmin.Handler(views.LogoutHandler))
	http.HandleFunc("/", uadmin.Handler(views.RootHandler))

	uadmin.RootURL = "/admin/"
	uadmin.Port = config.Port
}
