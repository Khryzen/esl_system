package models

import (
	"errors"
	"testing"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

func TestTeacherCredentialDelete(t *testing.T) {
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
		Title:     "TESOL Certificate",
	}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	if err := credential.Delete(); err != nil {
		t.Fatalf("Delete() returned an error: %v", err)
	}

	var saved TeacherCredential
	err := db.First(&saved, credential.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("credential lookup error = %v, want %v",
			err, gorm.ErrRecordNotFound)
	}
}

func TestTeacherCredentialDeleteRejectsMissingID(t *testing.T) {
	setupTeacherProfileTestDB(t)

	credential := TeacherCredential{
		Title: "TESOL Certificate",
	}

	if err := credential.Delete(); err == nil {
		t.Fatal("Delete() returned nil, want an error for ID zero")
	}
}

func TestTeacherCredentialDeleteRejectsMissingCredential(t *testing.T) {
	setupTeacherProfileTestDB(t)

	credential := TeacherCredential{
		Model: uadmin.Model{ID: 999999},
		Title: "TESOL Certificate",
	}

	err := credential.Delete()
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Delete() error = %v, want %v",
			err, gorm.ErrRecordNotFound)
	}
}

func TestTeacherCredentialDeleteDoesNotDeleteOtherCredentials(t *testing.T) {
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
	}
	second := TeacherCredential{
		TeacherID: teacher.ID,
		Title:     "Teaching Diploma",
	}

	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("failed to create first credential: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("failed to create second credential: %v", err)
	}

	if err := first.Delete(); err != nil {
		t.Fatalf("Delete() returned an error: %v", err)
	}

	var remaining TeacherCredential
	if err := db.First(&remaining, second.ID).Error; err != nil {
		t.Fatalf("second credential should remain: %v", err)
	}

	if remaining.Title != second.Title {
		t.Errorf("remaining title = %q, want %q",
			remaining.Title, second.Title)
	}

	var count int64
	if err := db.Model(&TeacherCredential{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count credentials: %v", err)
	}
	if count != 1 {
		t.Errorf("remaining credential count = %d, want 1", count)
	}
}
