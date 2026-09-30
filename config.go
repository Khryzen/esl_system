package main

import (
	"errors"
	"fmt"
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

func (c *AppConfig) IsConfigured() bool {
	if c.Environment == "" {
		return false
	}

	if c.Environment == "PRODUCTION" {
		return c.Database != nil
	}

	return true
}

func (c AppConfig) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		uadmin.Trail(uadmin.CRITICAL, "invalid server port: %d", c.Port)
		return fmt.Errorf("invalid server port: %d", c.Port)
	}

	if c.Environment == "PRODUCTION" {
		if c.Database == nil {
			return errors.New("production database configuration is required")
		}

		if c.Database.Host == "" {
			return errors.New("DB_HOST is required in production")
		}

		if c.Database.Name == "" {
			return errors.New("DB_NAME is required in production")
		}

		if c.Database.User == "" {
			return errors.New("DB_USER is required in production")
		}

		if c.Database.Password == "" {
			return errors.New("DB_PASSWORD is required in production")
		}

		if c.Database.Port < 1 || c.Database.Port > 65535 {
			return fmt.Errorf("invalid database port: %d", c.Database.Port)
		}
	}

	return nil
}
