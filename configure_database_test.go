package main

import (
	"testing"

	"github.com/uadmin/uadmin"
)

func TestConfigureDatabase(t *testing.T) {
	t.Run("sets uadmin database when configured", func(t *testing.T) {
		database := &uadmin.DBSettings{
			Host:     "localhost",
			Name:     "esl",
			User:     "postgres",
			Password: "secret",
			Port:     5432,
			Type:     "postgres",
		}

		config := AppConfig{
			Database: database,
		}

		original := uadmin.Database
		t.Cleanup(func() {
			uadmin.Database = original
		})

		configureDatabase(config)

		if uadmin.Database != database {
			t.Fatal("expected uadmin.Database to use the configured database")
		}
	})

	t.Run("does not change uadmin database when not configured", func(t *testing.T) {
		original := &uadmin.DBSettings{
			Host: "existing-host",
		}

		uadmin.Database = original
		db := uadmin.GetDB()
		t.Cleanup(func() {
			if db != nil {
				if sqlDB, err := db.DB(); err == nil {
					_ = sqlDB.Close()
				}
			}

			uadmin.ClearDB()
			uadmin.Database = nil
		})

		config := AppConfig{}

		configureDatabase(config)

		if uadmin.Database != original {
			t.Fatal("expected existing uadmin.Database to remain unchanged")
		}
	})
}
