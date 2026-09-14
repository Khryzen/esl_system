package models

import "github.com/uadmin/uadmin"

type Enrollment struct {
	uadmin.Model
	Student          Student
	StudentID        uint
	Course           Course
	CourseID         uint
	Package          Package
	PackageID        uint
	TotalClasses     int
	ClassesRemaining int
	Active           bool
}
