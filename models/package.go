package models

import (
	"time"

	"github.com/uadmin/uadmin"
)

type Package struct {
	uadmin.Model
	Name                     string     `uadmin:"required"`
	NumberOfClasses          int        `uadmin:"required"`
	NumberOfFreeClasses      int        `uadmin:"required"`
	ClassDurationInMinutes   int        `uadmin:"required"`
	Price                    float64    `uadmin:"required"`
	TimeBetweenClassesInDays int        `uadmin:"required"`
	ValidFrom                *time.Time `uadmin:"required"`
	ValidUntil               *time.Time `uadmin:"required"`
	Active                   bool       `uadmin:"required"`
}
