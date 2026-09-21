package models

import (
	"time"

	"github.com/uadmin/uadmin"
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
