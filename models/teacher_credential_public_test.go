package models

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestGetPublicTeacherCredentials(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	teacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}
	if err := db.Create(&teacher).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	publicCredential := TeacherCredential{
		TeacherID: teacher.ID,
		Title:     "TESOL Certificate",
		IsPublic:  true,
	}
	privateCredential := TeacherCredential{
		TeacherID: teacher.ID,
		Title:     "Private Qualification",
		IsPublic:  false,
	}

	if err := db.Create(&publicCredential).Error; err != nil {
		t.Fatalf("failed to create public credential: %v", err)
	}
	if err := db.Create(&privateCredential).Error; err != nil {
		t.Fatalf("failed to create private credential: %v", err)
	}

	credentials, err := GetPublicTeacherCredentials(teacher.ID)
	if err != nil {
		t.Fatalf("GetPublicTeacherCredentials() returned an error: %v", err)
	}

	if len(credentials) != 1 {
		t.Fatalf("got %d credentials, want 1", len(credentials))
	}

	if credentials[0].ID != publicCredential.ID {
		t.Errorf("credential ID = %d, want %d",
			credentials[0].ID, publicCredential.ID)
	}
	if !credentials[0].IsPublic {
		t.Error("returned credential is not public")
	}
}

func TestGetPublicTeacherCredentialsEmpty(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	teacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}
	if err := db.Create(&teacher).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	privateCredential := TeacherCredential{
		TeacherID: teacher.ID,
		Title:     "Private Qualification",
		IsPublic:  false,
	}
	if err := db.Create(&privateCredential).Error; err != nil {
		t.Fatalf("failed to create private credential: %v", err)
	}

	credentials, err := GetPublicTeacherCredentials(teacher.ID)
	if err != nil {
		t.Fatalf("GetPublicTeacherCredentials() returned an error: %v", err)
	}

	if len(credentials) != 0 {
		t.Errorf("got %d credentials, want 0", len(credentials))
	}
}

func TestGetPublicTeacherCredentialsMissingTeacher(t *testing.T) {
	setupTeacherProfileTestDB(t)

	credentials, err := GetPublicTeacherCredentials(999999)

	if credentials != nil {
		t.Errorf("credentials = %v, want nil", credentials)
	}
	if !errors.Is(err, ErrTeacherNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrTeacherNotFound)
	}
}

func TestGetPublicTeacherCredentialsZeroID(t *testing.T) {
	setupTeacherProfileTestDB(t)

	credentials, err := GetPublicTeacherCredentials(0)

	if credentials != nil {
		t.Errorf("credentials = %v, want nil", credentials)
	}
	if !errors.Is(err, ErrTeacherNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrTeacherNotFound)
	}
}

func TestPublicCredentialVisibilityChangesAfterUpdate(t *testing.T) {
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
		IsPublic:  false,
	}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	publicCredentials, err := GetPublicTeacherCredentials(teacher.ID)
	if err != nil {
		t.Fatalf("initial retrieval failed: %v", err)
	}
	if len(publicCredentials) != 0 {
		t.Fatalf("initial public credential count = %d, want 0",
			len(publicCredentials))
	}

	credential.IsPublic = true
	if err := credential.Update(); err != nil {
		t.Fatalf("credential Update() returned an error: %v", err)
	}

	publicCredentials, err = GetPublicTeacherCredentials(teacher.ID)
	if err != nil {
		t.Fatalf("retrieval after update failed: %v", err)
	}
	if len(publicCredentials) != 1 {
		t.Fatalf("public credential count after update = %d, want 1",
			len(publicCredentials))
	}
	if publicCredentials[0].ID != credential.ID {
		t.Errorf("returned credential ID = %d, want %d",
			publicCredentials[0].ID, credential.ID)
	}
}

func TestGetPublicTeacherCredentialsDoesNotReturnOtherTeachersCredentials(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	firstTeacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}
	secondTeacher := Teacher{
		FirstName: "Jamie",
		LastName:  "Cruz",
	}

	if err := db.Create(&firstTeacher).Error; err != nil {
		t.Fatalf("failed to create first teacher: %v", err)
	}
	if err := db.Create(&secondTeacher).Error; err != nil {
		t.Fatalf("failed to create second teacher: %v", err)
	}

	credential := TeacherCredential{
		TeacherID: secondTeacher.ID,
		Title:     "Other Teacher Certificate",
		IsPublic:  true,
	}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	credentials, err := GetPublicTeacherCredentials(firstTeacher.ID)
	if err != nil {
		t.Fatalf("GetPublicTeacherCredentials() returned an error: %v", err)
	}

	if len(credentials) != 0 {
		t.Errorf("got %d credentials, want 0", len(credentials))
	}

	if err := db.First(&TeacherCredential{}, credential.ID).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unexpected credential lookup error: %v", err)
	}
}
