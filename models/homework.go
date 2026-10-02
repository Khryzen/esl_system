package models

import "github.com/uadmin/uadmin"

type Homework struct {
	uadmin.Model
	Assessment   Assessment
	AssessmentID uint `gorm:"uniqueIndex"`
	Title        string
	HomeworkFile string
}
