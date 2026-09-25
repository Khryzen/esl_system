package main

import (
	"net/http"
	"os"
	"strconv"

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

	appPort, err := strconv.Atoi(os.Getenv("UADMIN_PORT"))
	if err != nil {
		appPort = 1123
	}

	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
		if err != nil {
			dbPort = 5432
		}

		uadmin.Database = &uadmin.DBSettings{
			Host:     os.Getenv("DB_HOST"),
			Name:     os.Getenv("DB_NAME"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Port:     dbPort,
			Type:     "postgres",
		}
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
	uadmin.Port = appPort
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
