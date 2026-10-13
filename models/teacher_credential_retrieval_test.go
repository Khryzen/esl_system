package models

import (
	"errors"
	"testing"

	"github.com/uadmin/uadmin"
)

func TestGetTeacherCredentials(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	teacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}
	if err := db.Create(&teacher).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	first := TeacherCredential{
		TeacherID: teacher.ID,
		Title:     "TESOL Certificate",
		IsPublic:  true,
	}
	second := TeacherCredential{
		TeacherID: teacher.ID,
		Title:     "Teaching Diploma",
		IsPublic:  false,
	}

	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("failed to create first credential: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("failed to create second credential: %v", err)
	}

	credentials, err := GetTeacherCredentials(teacher.ID)
	if err != nil {
		t.Fatalf("GetTeacherCredentials() returned an error: %v", err)
	}

	if len(credentials) != 2 {
		t.Fatalf("got %d credentials, want 2", len(credentials))
	}

	if credentials[0].ID != first.ID {
		t.Errorf("first credential ID = %d, want %d",
			credentials[0].ID, first.ID)
	}
	if credentials[1].ID != second.ID {
		t.Errorf("second credential ID = %d, want %d",
			credentials[1].ID, second.ID)
	}
}

func TestGetTeacherCredentialsEmpty(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	teacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}
	if err := db.Create(&teacher).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	credentials, err := GetTeacherCredentials(teacher.ID)
	if err != nil {
		t.Fatalf("GetTeacherCredentials() returned an error: %v", err)
	}

	if len(credentials) != 0 {
		t.Errorf("got %d credentials, want 0", len(credentials))
	}
}

func TestGetTeacherCredentialsMissingTeacher(t *testing.T) {
	setupTeacherProfileTestDB(t)

	credentials, err := GetTeacherCredentials(999999)

	if credentials != nil {
		t.Errorf("credentials = %v, want nil", credentials)
	}
	if !errors.Is(err, ErrTeacherNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrTeacherNotFound)
	}
}

func TestGetTeacherCredentialsZeroID(t *testing.T) {
	setupTeacherProfileTestDB(t)

	credentials, err := GetTeacherCredentials(0)

	if credentials != nil {
		t.Errorf("credentials = %v, want nil", credentials)
	}
	if !errors.Is(err, ErrTeacherNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrTeacherNotFound)
	}
}

// Ensure the public visibility flag remains available for future filtering.
func TestTeacherCredentialVisibilityStored(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	teacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}
	if err := db.Create(&teacher).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	credential := TeacherCredential{
		TeacherID: teacher.ID,
		Title:     "Private Qualification",
		IsPublic:  false,
	}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	var saved TeacherCredential
	if err := db.First(&saved, credential.ID).Error; err != nil {
		t.Fatalf("failed to reload credential: %v", err)
	}

	if saved.IsPublic {
		t.Error("IsPublic = true, want false")
	}

	_ = uadmin.GetDB()
}
