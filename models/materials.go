package models

import (
	"context"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

type materialInternalSaveContextKey struct{}

type Material struct {
	uadmin.Model
	Name   string `uadmin:"required"`
	File   string `uadmin:"required"`
	Active bool   `uadmin:"required"`
}

func (m *Material) BeforeSave(tx *gorm.DB) error {
	if tx.Statement.Context.Value(materialInternalSaveContextKey{}) == true {
		return nil
	}

	if m.ID == 0 {
		return nil
	}

	var existing Material

	if err := tx.Unscoped().First(&existing, m.ID).Error; err != nil {
		return err
	}

	m.File = existing.File

	return nil
}

func withMaterialInternalSave(tx *gorm.DB) *gorm.DB {
	ctx := context.WithValue(
		tx.Statement.Context,
		materialInternalSaveContextKey{},
		true,
	)

	return tx.WithContext(ctx)
}
