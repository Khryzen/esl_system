package main

import (
	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func registerModels() {
	uadmin.Register(
		models.Assessment{},
		models.Class{},
		models.Course{},
		models.CourseMaterial{},
		models.Enrollment{},
		models.Homework{},
		models.Invoice{},
		models.Level{},
		models.Material{},
		models.Package{},
		models.Student{},
		models.Teacher{},
	)
}
