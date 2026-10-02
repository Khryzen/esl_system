package models

import (
	"testing"
	"time"

	"github.com/uadmin/uadmin"
)

func TestPackageValidate(t *testing.T) {
	t.Run("accepts positive class duration", func(t *testing.T) {
		pkg := Package{
			ClassDurationInMinutes: 60,
		}

		errors := pkg.Validate()

		if len(errors) != 0 {
			t.Fatalf("Validate() errors = %v, want none", errors)
		}
	})

	t.Run("rejects zero class duration", func(t *testing.T) {
		pkg := Package{
			ClassDurationInMinutes: 0,
		}

		errors := pkg.Validate()

		message, ok := errors["ClassDurationInMinutes"]
		if !ok {
			t.Fatal("expected ClassDurationInMinutes validation error")
		}

		if message != ErrPackageInvalidClassDuration.Error() {
			t.Fatalf(
				"validation error = %q, want %q",
				message,
				ErrPackageInvalidClassDuration.Error(),
			)
		}
	})

	t.Run("rejects negative class duration", func(t *testing.T) {
		pkg := Package{
			ClassDurationInMinutes: -30,
		}

		errors := pkg.Validate()

		message, ok := errors["ClassDurationInMinutes"]
		if !ok {
			t.Fatal("expected ClassDurationInMinutes validation error")
		}

		if message != ErrPackageInvalidClassDuration.Error() {
			t.Fatalf(
				"validation error = %q, want %q",
				message,
				ErrPackageInvalidClassDuration.Error(),
			)
		}
	})
}

func TestPackageSaveIntegrity(t *testing.T) {
	uadmin.ClearDB()
	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/package_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(&Package{}); err != nil {
		t.Fatalf("migrate package: %v", err)
	}

	validFrom := time.Now().Add(-24 * time.Hour)
	validUntil := time.Now().Add(30 * 24 * time.Hour)

	pkg := Package{
		Name:                   "Test Package",
		NumberOfClasses:        10,
		NumberOfFreeClasses:    2,
		ClassDurationInMinutes: 50,
		Price:                  1000,
		ValidFrom:              &validFrom,
		ValidUntil:             &validUntil,
		Active:                 true,
	}

	if err := db.Create(&pkg).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	pkg.TotalClasses = 999

	if err := db.Save(&pkg).Error; err != nil {
		t.Fatalf("save package: %v", err)
	}

	var saved Package
	if err := db.First(&saved, pkg.ID).Error; err != nil {
		t.Fatalf("reload package: %v", err)
	}

	if saved.TotalClasses != 12 {
		t.Fatalf(
			"TotalClasses = %d, want protected value 12",
			saved.TotalClasses,
		)
	}
}
