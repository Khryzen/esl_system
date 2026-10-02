package models

import (
	"errors"
	"time"

	"github.com/uadmin/uadmin"
)

var (
	ErrPackageInvalidClassDuration = errors.New(
		"The class duration must be greater than zero.",
	)
)

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
