package models

import (
	"errors"
	"testing"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

func setupTeacherProfileTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	uadmin.ClearDB()
	uadmin.Database = &uadmin.DBSettings{
		Type: "sqlite",
		Name: t.TempDir() + "/teacher_profile_test.db",
	}

	db := uadmin.GetDB()

	if err := db.AutoMigrate(
		&uadmin.User{},
		&Teacher{},
		&TeacherCredential{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	t.Cleanup(func() {
		cleanupTestDB(t, db)
	})

	return db
}

func TestTeacherValidation(t *testing.T) {
	tests := []struct {
		name    string
		teacher Teacher
		wantErr error
	}{
		{
			name: "valid teacher",
			teacher: Teacher{
				FirstName:       "Alex",
				LastName:        "Santos",
				YearsExperience: 5,
			},
		},
		{
			name: "missing first name",
			teacher: Teacher{
				LastName: "Santos",
			},
			wantErr: ErrTeacherFirstNameRequired,
		},
		{
			name: "blank first name",
			teacher: Teacher{
				FirstName: "   ",
				LastName:  "Santos",
			},
			wantErr: ErrTeacherFirstNameRequired,
		},
		{
			name: "missing last name",
			teacher: Teacher{
				FirstName: "Alex",
			},
			wantErr: ErrTeacherLastNameRequired,
		},
		{
			name: "negative experience",
			teacher: Teacher{
				FirstName:       "Alex",
				LastName:        "Santos",
				YearsExperience: -1,
			},
			wantErr: ErrTeacherInvalidExperience,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTeacher(&tt.teacher)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateTeacher() error = %v, want %v",
					err, tt.wantErr)
			}
		})
	}
}

func TestTeacherCredentialValidation(t *testing.T) {
	issueDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	expirationDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	earlierDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		credential TeacherCredential
		wantErr    error
	}{
		{
			name: "valid credential",
			credential: TeacherCredential{
				TeacherID:      1,
				Title:          "TESOL Certificate",
				IssueDate:      &issueDate,
				ExpirationDate: &expirationDate,
			},
		},
		{
			name: "missing title",
			credential: TeacherCredential{
				TeacherID: 1,
				Title:     " ",
			},
			wantErr: ErrCredentialTitleRequired,
		},
		{
			name: "missing teacher",
			credential: TeacherCredential{
				Title: "TESOL Certificate",
			},
			wantErr: ErrCredentialTeacherRequired,
		},
		{
			name: "expiration before issue date",
			credential: TeacherCredential{
				TeacherID:      1,
				Title:          "TESOL Certificate",
				IssueDate:      &issueDate,
				ExpirationDate: &earlierDate,
			},
			wantErr: ErrCredentialInvalidDates,
		},
		{
			name: "no expiration date",
			credential: TeacherCredential{
				TeacherID: 1,
				Title:     "Lifetime Certificate",
				IssueDate: &issueDate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTeacherCredential(&tt.credential)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"validateTeacherCredential() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestTeacherCredentialPersistence(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	teacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}

	if err := db.Create(&teacher).Error; err != nil {
		t.Fatalf("failed to create teacher: %v", err)
	}

	credential := TeacherCredential{
		TeacherID:           teacher.ID,
		Title:               "TESOL Certificate",
		IssuingOrganization: "Example Institute",
		CredentialType:      "Certificate",
		IsPublic:            true,
	}

	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	var saved TeacherCredential
	if err := db.First(&saved, credential.ID).Error; err != nil {
		t.Fatalf("failed to reload credential: %v", err)
	}

	if saved.TeacherID != teacher.ID {
		t.Fatalf("TeacherID = %d, want %d", saved.TeacherID, teacher.ID)
	}

	if saved.Title != credential.Title {
		t.Fatalf("Title = %q, want %q", saved.Title, credential.Title)
	}
}

func TestTeacherCredentialRejectsMissingTeacher(t *testing.T) {
	setupTeacherProfileTestDB(t)

	credential := TeacherCredential{
		TeacherID: 999999,
		Title:     "TESOL Certificate",
	}

	err := uadmin.GetDB().Create(&credential).Error

	if !errors.Is(err, ErrCredentialTeacherNotFound) {
		t.Fatalf("Create() error = %v, want %v",
			err, ErrCredentialTeacherNotFound)
	}
}

func TestTeacherCredentialProtectsTeacherID(t *testing.T) {
	db := setupTeacherProfileTestDB(t)

	firstTeacher := Teacher{
		FirstName: "Alex",
		LastName:  "Santos",
	}
	if err := db.Create(&firstTeacher).Error; err != nil {
		t.Fatalf("failed to create first teacher: %v", err)
	}

	secondTeacher := Teacher{
		FirstName: "Jamie",
		LastName:  "Cruz",
	}
	if err := db.Create(&secondTeacher).Error; err != nil {
		t.Fatalf("failed to create second teacher: %v", err)
	}

	credential := TeacherCredential{
		TeacherID: firstTeacher.ID,
		Title:     "TESOL Certificate",
	}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	credential.TeacherID = secondTeacher.ID
	if err := db.Save(&credential).Error; err != nil {
		t.Fatalf("failed to save credential: %v", err)
	}

	var saved TeacherCredential
	if err := db.First(&saved, credential.ID).Error; err != nil {
		t.Fatalf("failed to reload credential: %v", err)
	}

	if saved.TeacherID != firstTeacher.ID {
		t.Fatalf("TeacherID = %d, want protected value %d",
			saved.TeacherID, firstTeacher.ID)
	}
}
