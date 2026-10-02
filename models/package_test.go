package models

import (
	"testing"
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
