package models

import (
	"context"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

type courseMaterialInternalSaveContextKey struct{}

type CourseMaterial struct {
	uadmin.Model
	Course     Course
	CourseID   uint
	Material   Material
	MaterialID uint
	Active     bool `uadmin:"required"`
}

func (cm *CourseMaterial) BeforeSave(tx *gorm.DB) error {
	if tx.Statement.Context.Value(courseMaterialInternalSaveContextKey{}) == true {
		return nil
	}

	if cm.ID == 0 {
		return nil
	}

	var existing CourseMaterial

	if err := tx.Unscoped().First(&existing, cm.ID).Error; err != nil {
		return err
	}

	cm.CourseID = existing.CourseID
	cm.MaterialID = existing.MaterialID

	return nil
}

func withCourseMaterialInternalSave(tx *gorm.DB) *gorm.DB {
	ctx := context.WithValue(
		tx.Statement.Context,
		courseMaterialInternalSaveContextKey{},
		true,
	)

	return tx.WithContext(ctx)
}
