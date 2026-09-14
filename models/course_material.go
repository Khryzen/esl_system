package models

import "github.com/uadmin/uadmin"

type CourseMaterial struct {
	uadmin.Model
	Course     Course
	CourseID   uint
	Material   Material
	MaterialID uint
	Active     bool `uadmin:"required"`
}
