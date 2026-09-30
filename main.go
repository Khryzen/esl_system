package main

import (
	"net/http"

	"github.com/Khryzen/esl_system/models"
	"github.com/Khryzen/esl_system/views"
	"github.com/joho/godotenv"
	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

var DB *gorm.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		uadmin.Trail(uadmin.ERROR, "Error loading environment variables: %v", err)
		return
	}

	config := loadAppConfig()

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

	InitialData()

	http.HandleFunc("/login/", uadmin.Handler(views.LoginHandler))
	http.HandleFunc("/logout/", uadmin.Handler(views.LogoutHandler))
	http.HandleFunc("/", uadmin.Handler(views.RootHandler))

	uadmin.RootURL = "/admin/"
	uadmin.Port = config.Port
	uadmin.StartServer()
}

func InitialData() {
	levelData := []models.Level{
		{
			Level: "Newbie",
		},
		{
			Level: "Beginner",
		},
		{
			Level: "Intermediate",
		},
		{
			Level: "Moderate",
		},
		{
			Level: "Proficient",
		},
	}

	level := models.Level{}
	if uadmin.Count(&level, "id > 0") == 0 {
		for i := range levelData {
			uadmin.Save(&levelData[i])
		}
	}
}
