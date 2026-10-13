package models

import (
	"errors"
	"testing"

	"github.com/uadmin/uadmin"
)

func TestGetTeacherProfile(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	expected := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
		Email:     "alex@example.com",
	}

	if err := db.Create(&expected).Error; err != nil {
		t.Fatalf("failed to create teacher profile: %v", err)
	}

	got, err := GetTeacherProfile()
	if err != nil {
		t.Fatalf("GetTeacherProfile() returned an error: %v", err)
	}

	if got.ID != expected.ID {
		t.Errorf("ID = %d, want %d", got.ID, expected.ID)
	}

	if got.FirstName != expected.FirstName {
		t.Errorf("FirstName = %q, want %q", got.FirstName, expected.FirstName)
	}

	if got.LastName != expected.LastName {
		t.Errorf("LastName = %q, want %q", got.LastName, expected.LastName)
	}

	if got.Email != expected.Email {
		t.Errorf("Email = %q, want %q", got.Email, expected.Email)
	}

}

func TestGetTeacherProfileNotFound(t *testing.T) {
	setupTeacherProfileTestDB(t)

	got, err := GetTeacherProfile()

	if got != nil {
		t.Fatalf("GetTeacherProfile() returned a profile, want nil")
	}

	if !errors.Is(err, ErrTeacherNotFound) {
		t.Fatalf("GetTeacherProfile() error = %v, want %v",
			err, ErrTeacherNotFound)
	}

}

func TestGetTeacherProfileDoesNotCreateProfile(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	_, err := GetTeacherProfile()
	if !errors.Is(err, ErrTeacherNotFound) {
		t.Fatalf("GetTeacherProfile() error = %v, want %v",
			err, ErrTeacherNotFound)
	}

	var count int64
	if err := db.Model(&Teacher{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count teacher profiles: %v", err)
	}

	if count != 0 {
		t.Errorf("teacher profile count = %d, want 0", count)
	}

	_ = uadmin.GetDB()

}
