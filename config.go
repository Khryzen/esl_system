package main

import (
	"os"
	"strconv"

	"github.com/uadmin/uadmin"
)

type AppConfig struct {
	Port        int
	Environment string
	Database    *uadmin.DBSettings
}

func loadAppConfig() AppConfig {
	config := AppConfig{
		Port:        1123,
		Environment: os.Getenv("ENVIRONMENT"),
	}

	if port, err := strconv.Atoi(os.Getenv("UADMIN_PORT")); err == nil {
		config.Port = port
	}

	if config.Environment == "PRODUCTION" {
		dbPort := 5432

		if port, err := strconv.Atoi(os.Getenv("DB_PORT")); err == nil {
			dbPort = port
		}

		config.Database = &uadmin.DBSettings{
			Host:     os.Getenv("DB_HOST"),
			Name:     os.Getenv("DB_NAME"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Port:     dbPort,
			Type:     "postgres",
		}
	}

	return config
}
