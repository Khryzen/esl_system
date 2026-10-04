package views

import (
	"testing"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

func cleanupTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}

	uadmin.ClearDB()
	uadmin.Database = nil
}
