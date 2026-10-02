package models

import (
	"context"
	"errors"
	"time"

	"github.com/uadmin/uadmin"
	"gorm.io/gorm"
)

var (
	ErrPackageInvalidClassDuration = errors.New(
		"The class duration must be greater than zero.",
	)
)

type packageInternalSaveContextKey struct{}

type Package struct {
	uadmin.Model
	Name                   string `uadmin:"required"`
	NumberOfClasses        int    `uadmin:"required"`
	NumberOfFreeClasses    int    `uadmin:"required"`
	TotalClasses           int
	ClassDurationInMinutes int        `uadmin:"required"`
	Price                  float64    `uadmin:"required"`
	ValidFrom              *time.Time `uadmin:"required"`
	ValidUntil             *time.Time `uadmin:"required"`
	Image                  string
	Active                 bool `uadmin:"required"`
}

func (p Package) String() string {
	return p.Name
}

func (p *Package) BeforeSave(tx *gorm.DB) error {
	// TotalClasses is derived from the two class-count fields.
	//
	// Internal saves are allowed to set it explicitly after calculating
	// the derived value.
	if tx.Statement.Context.Value(packageInternalSaveContextKey{}) == true {
		return nil
	}

	// For new packages, always derive TotalClasses from its source fields.
	if p.ID == 0 {
		p.TotalClasses = p.NumberOfClasses + p.NumberOfFreeClasses
		return nil
	}

	// For existing packages, prevent generic saves from modifying the
	// derived field independently.
	var existing Package

	if err := tx.Unscoped().First(&existing, p.ID).Error; err != nil {
		return err
	}

	p.TotalClasses = existing.TotalClasses

	return nil
}

func withPackageInternalSave(tx *gorm.DB) *gorm.DB {
	ctx := context.WithValue(
		tx.Statement.Context,
		packageInternalSaveContextKey{},
		true,
	)

	return tx.WithContext(ctx)
}

func (p *Package) Save() {
	p.TotalClasses = p.NumberOfClasses + p.NumberOfFreeClasses
	uadmin.Save(p)
}

func (p Package) Validate() (ret map[string]string) {
	ret = map[string]string{}

	if p.ClassDurationInMinutes <= 0 {
		ret["ClassDurationInMinutes"] = ErrPackageInvalidClassDuration.Error()
	}

	return
}
