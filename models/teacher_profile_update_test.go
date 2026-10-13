package models

import (
	"errors"
	"testing"

	"github.com/uadmin/uadmin"
)

func TestTeacherUpdate(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	original := Teacher{
		FirstName:       "Alex",
		LastName:        "Santos",
		UserID:          42,
		Email:           "old@example.com",
		Phone:           "09170000000",
		Bio:             "Original bio",
		Languages:       "English",
		Specializations: "General English",
		Education:       "Bachelor's degree",
		YearsExperience: 3,
	}

	if err := db.Create(&original).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	updated := Teacher{
		Model:           original.Model,
		FirstName:       " Jamie ",
		LastName:        " Cruz ",
		UserID:          999,
		Email:           " jamie@example.com ",
		Phone:           "09171111111",
		ProfilePhoto:    "/uploads/jamie.jpg",
		Bio:             "Updated bio",
		Languages:       "English, Filipino",
		Specializations: "Business English",
		Education:       "Master's degree",
		YearsExperience: 5,
	}

	if err := updated.Update(); err != nil {
		t.Fatalf("Update() returned an error: %v", err)
	}

	if updated.ID != original.ID {
		t.Errorf("ID = %d, want %d", updated.ID, original.ID)
	}

	if updated.UserID != original.UserID {
		t.Errorf("UserID = %d, want %d", updated.UserID, original.UserID)
	}

	if updated.FirstName != "Jamie" {
		t.Errorf("FirstName = %q, want %q", updated.FirstName, "Jamie")
	}

	if updated.LastName != "Cruz" {
		t.Errorf("LastName = %q, want %q", updated.LastName, "Cruz")
	}

	var saved Teacher
	if err := db.First(&saved, original.ID).Error; err != nil {
		t.Fatalf("failed to reload teacher: %v", err)
	}

	if saved.Email != "jamie@example.com" {
		t.Errorf("Email = %q, want %q", saved.Email, "jamie@example.com")
	}

	if saved.Phone != updated.Phone {
		t.Errorf("Phone = %q, want %q", saved.Phone, updated.Phone)
	}

	if saved.Bio != updated.Bio {
		t.Errorf("Bio = %q, want %q", saved.Bio, updated.Bio)
	}

	if saved.YearsExperience != 5 {
		t.Errorf("YearsExperience = %d, want 5", saved.YearsExperience)
	}

	if saved.UserID != original.UserID {
		t.Errorf("persisted UserID = %d, want %d", saved.UserID, original.UserID)
	}
}

func TestTeacherUpdateRejectsInvalidProfile(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	original := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
		Email:     "alex@example.com",
	}

	if err := db.Create(&original).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	updated := original
	updated.FirstName = "   "

	err := updated.Update()
	if !errors.Is(err, ErrTeacherFirstNameRequired) {
		t.Fatalf("Update() error = %v, want %v",
			err, ErrTeacherFirstNameRequired)
	}

	var saved Teacher
	if err := db.First(&saved, original.ID).Error; err != nil {
		t.Fatalf("failed to reload teacher: %v", err)
	}

	if saved.FirstName != original.FirstName {
		t.Errorf("FirstName = %q, want unchanged value %q",
			saved.FirstName, original.FirstName)
	}

	if saved.Email != original.Email {
		t.Errorf("Email = %q, want unchanged value %q",
			saved.Email, original.Email)
	}
}

func TestTeacherUpdateRejectsNegativeExperience(t *testing.T) {
	setupTeacherProfileTestDB(t)

	teacher := Teacher{
		Model:           uadmin.Model{ID: 1},
		FirstName:       "Alex",
		LastName:        "Santos",
		YearsExperience: -1,
	}

	err := teacher.Update()
	if !errors.Is(err, ErrTeacherInvalidExperience) {
		t.Fatalf("Update() error = %v, want %v",
			err, ErrTeacherInvalidExperience)
	}
}

func TestTeacherUpdateRejectsMissingID(t *testing.T) {
	setupTeacherProfileTestDB(t)

	teacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}

	err := teacher.Update()
	if !errors.Is(err, ErrTeacherProfileIDRequired) {
		t.Fatalf("Update() error = %v, want %v",
			err, ErrTeacherProfileIDRequired)
	}
}

func TestTeacherUpdateRejectsMissingProfile(t *testing.T) {
	setupTeacherProfileTestDB(t)

	teacher := Teacher{
		Model:     uadmin.Model{ID: 999999},
		FirstName: "Alex",
		LastName:  "Santos",
	}

	err := teacher.Update()
	if !errors.Is(err, ErrTeacherNotFound) {
		t.Fatalf("Update() error = %v, want %v",
			err, ErrTeacherNotFound)
	}
}
