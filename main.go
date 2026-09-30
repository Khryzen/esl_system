package main

import (
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
	if err := config.Validate(); err != nil {
		uadmin.Trail(uadmin.CRITICAL, "Invalid application configuration: %v", err)
		return
	}

	setupApplication(config)
	initializeData()
	uadmin.StartServer()
}
