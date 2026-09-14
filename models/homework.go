package models

import "github.com/uadmin/uadmin"

type Homework struct {
	uadmin.Model
	Assessment   Assessment
	AssessmentID uint
	Title        string
	HomeworkFile string
}

// TODO: Create a helper function for the file
