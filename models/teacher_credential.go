package models

import (
	"time"

	"github.com/uadmin/uadmin"
)

type TeacherCredential struct {
	uadmin.Model

	Teacher             Teacher
	TeacherID           uint
	Title               string `uadmin:"required"`
	IssuingOrganization string
	CredentialType      string
	Description         string
	IssueDate           *time.Time
	ExpirationDate      *time.Time
	DocumentPath        string
	IsPublic            bool
}
