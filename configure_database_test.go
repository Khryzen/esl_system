package main

import (
	"testing"

	"github.com/uadmin/uadmin"
)

func TestConfigureDatabase(t *testing.T) {
	originalDatabase := uadmin.Database
	defer func() {
		uadmin.Database = originalDatabase
	}()

	t.Run("configures database when provided", func(t *testing.T) {
		database := &uadmin.DBSettings{
			Host:     "localhost",
			Name:     "testdb",
			User:     "testuser",
			Password: "testpass",
			Port:     5432,
			Type:     "postgres",
		}

		uadmin.Database = nil

		configureDatabase(AppConfig{
			Database: database,
		})

		if uadmin.Database != database {
			t.Fatal("expected uadmin.Database to use the configured database")
		}
	})

	t.Run("does not change database when not provided", func(t *testing.T) {
		existingDatabase := &uadmin.DBSettings{
			Host: "existing-host",
			Port: 5432,
			Type: "postgres",
		}

		uadmin.Database = existingDatabase

		configureDatabase(AppConfig{})

		if uadmin.Database != existingDatabase {
			t.Fatal("expected existing uadmin.Database to remain unchanged")
		}
	})
}
