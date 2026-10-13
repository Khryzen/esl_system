package models

import (
	"errors"
	"testing"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

func TestTeacherCredentialCreate(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	teacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}
	if err := db.Create(&teacher).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	issueDate := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	expirationDate := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)

	credential := TeacherCredential{
		TeacherID:           teacher.ID,
		Title:               "TESOL Certificate",
		IssuingOrganization: "Example Institute",
		CredentialType:      "Certificate",
		Description:         "Teaching English to speakers of other languages",
		IssueDate:           &issueDate,
		ExpirationDate:      &expirationDate,
		DocumentPath:        "/uploads/tesol.pdf",
		IsPublic:            true,
	}

	if err := credential.Create(); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	if credential.ID == 0 {
		t.Fatal("Create() did not assign an ID")
	}

	var saved TeacherCredential
	if err := db.First(&saved, credential.ID).Error; err != nil {
		t.Fatalf("failed to retrieve created credential: %v", err)
	}

	if saved.TeacherID != teacher.ID {
		t.Errorf("TeacherID = %d, want %d", saved.TeacherID, teacher.ID)
	}
	if saved.Title != credential.Title {
		t.Errorf("Title = %q, want %q", saved.Title, credential.Title)
	}
	if saved.IssuingOrganization != credential.IssuingOrganization {
		t.Errorf("IssuingOrganization = %q, want %q",
			saved.IssuingOrganization, credential.IssuingOrganization)
	}
	if saved.DocumentPath != credential.DocumentPath {
		t.Errorf("DocumentPath = %q, want %q",
			saved.DocumentPath, credential.DocumentPath)
	}
	if !saved.IsPublic {
		t.Error("IsPublic = false, want true")
	}
}

func TestTeacherCredentialCreateRejectsBlankTitle(t *testing.T) {
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
		Title:     "   ",
	}

	err := credential.Create()
	if !errors.Is(err, ErrCredentialTitleRequired) {
		t.Fatalf("Create() error = %v, want %v",
			err, ErrCredentialTitleRequired)
	}

	var count int64
	if err := db.Model(&TeacherCredential{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count credentials: %v", err)
	}
	if count != 0 {
		t.Errorf("credential count = %d, want 0", count)
	}
}

func TestTeacherCredentialCreateRejectsMissingTeacher(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	credential := TeacherCredential{
		TeacherID: 999999,
		Title:     "TESOL Certificate",
	}

	err := credential.Create()
	if !errors.Is(err, ErrCredentialTeacherNotFound) {
		t.Fatalf("Create() error = %v, want %v",
			err, ErrCredentialTeacherNotFound)
	}

	var count int64
	if err := db.Model(&TeacherCredential{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count credentials: %v", err)
	}
	if count != 0 {
		t.Errorf("credential count = %d, want 0", count)
	}
}

func TestTeacherCredentialCreateRejectsInvalidDates(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	teacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}
	if err := db.Create(&teacher).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	issueDate := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	expirationDate := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	credential := TeacherCredential{
		TeacherID:      teacher.ID,
		Title:          "TESOL Certificate",
		IssueDate:      &issueDate,
		ExpirationDate: &expirationDate,
	}

	err := credential.Create()
	if !errors.Is(err, ErrCredentialInvalidDates) {
		t.Fatalf("Create() error = %v, want %v",
			err, ErrCredentialInvalidDates)
	}

	var count int64
	if err := db.Model(&TeacherCredential{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count credentials: %v", err)
	}
	if count != 0 {
		t.Errorf("credential count = %d, want 0", count)
	}
}

func TestTeacherCredentialUpdate(t *testing.T) {
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
		Title:     "Original Certificate",
		IsPublic:  false,
	}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	updated := credential
	updated.Title = "Updated TESOL Certificate"
	updated.IssuingOrganization = "Example Institute"
	updated.CredentialType = "Professional Certificate"
	updated.Description = "Updated description"
	updated.DocumentPath = "/uploads/updated-tesol.pdf"
	updated.IsPublic = true

	// Attempt to change ownership; Update must preserve the stored TeacherID.
	updated.TeacherID = teacher.ID + 1000

	if err := updated.Update(); err != nil {
		t.Fatalf("Update() returned an error: %v", err)
	}

	if updated.TeacherID != teacher.ID {
		t.Errorf("TeacherID = %d, want original teacher ID %d",
			updated.TeacherID, teacher.ID)
	}

	var saved TeacherCredential
	if err := db.First(&saved, credential.ID).Error; err != nil {
		t.Fatalf("failed to reload credential: %v", err)
	}

	if saved.TeacherID != teacher.ID {
		t.Errorf("persisted TeacherID = %d, want %d",
			saved.TeacherID, teacher.ID)
	}
	if saved.Title != "Updated TESOL Certificate" {
		t.Errorf("Title = %q, want %q",
			saved.Title, "Updated TESOL Certificate")
	}
	if saved.IssuingOrganization != "Example Institute" {
		t.Errorf("IssuingOrganization = %q, want %q",
			saved.IssuingOrganization, "Example Institute")
	}
	if saved.CredentialType != "Professional Certificate" {
		t.Errorf("CredentialType = %q, want %q",
			saved.CredentialType, "Professional Certificate")
	}
	if saved.Description != "Updated description" {
		t.Errorf("Description = %q, want %q",
			saved.Description, "Updated description")
	}
	if saved.DocumentPath != "/uploads/updated-tesol.pdf" {
		t.Errorf("DocumentPath = %q, want %q",
			saved.DocumentPath, "/uploads/updated-tesol.pdf")
	}
	if !saved.IsPublic {
		t.Error("IsPublic = false, want true after update")
	}
}

func TestTeacherCredentialUpdateRejectsBlankTitle(t *testing.T) {
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
		Title:     "Original Certificate",
	}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	updated := credential
	updated.Title = "   "

	err := updated.Update()
	if !errors.Is(err, ErrCredentialTitleRequired) {
		t.Fatalf("Update() error = %v, want %v",
			err, ErrCredentialTitleRequired)
	}

	var saved TeacherCredential
	if err := db.First(&saved, credential.ID).Error; err != nil {
		t.Fatalf("failed to reload credential: %v", err)
	}
	if saved.Title != credential.Title {
		t.Errorf("Title = %q, want unchanged value %q",
			saved.Title, credential.Title)
	}
}

func TestTeacherCredentialUpdateRejectsInvalidDates(t *testing.T) {
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
		Title:     "Original Certificate",
	}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	issueDate := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	expirationDate := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	updated := credential
	updated.IssueDate = &issueDate
	updated.ExpirationDate = &expirationDate

	err := updated.Update()
	if !errors.Is(err, ErrCredentialInvalidDates) {
		t.Fatalf("Update() error = %v, want %v",
			err, ErrCredentialInvalidDates)
	}

	var saved TeacherCredential
	if err := db.First(&saved, credential.ID).Error; err != nil {
		t.Fatalf("failed to reload credential: %v", err)
	}
	if saved.IssueDate != nil || saved.ExpirationDate != nil {
		t.Error("invalid dates were persisted")
	}
}

func TestTeacherCredentialUpdateRejectsMissingCredential(t *testing.T) {
	setupTeacherProfileTestDB(t)

	credential := TeacherCredential{
		Model:     uadmin.Model{ID: 999999},
		TeacherID: 1,
		Title:     "TESOL Certificate",
	}

	err := credential.Update()
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Update() error = %v, want %v",
			err, gorm.ErrRecordNotFound)
	}
}

func TestTeacherCredentialCreateRejectsExistingID(t *testing.T) {
	setupTeacherProfileTestDB(t)

	credential := TeacherCredential{
		Model: uadmin.Model{ID: 1},
		Title: "TESOL Certificate",
	}

	err := credential.Create()
	if err == nil {
		t.Fatal("Create() error = nil, want an error for an existing ID")
	}
}
