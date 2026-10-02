package models

import (
	"time"

	"github.com/uadmin/uadmin"
)

type Assessment struct {
	uadmin.Model
	Date               time.Time
	Class              Class
	ClassID            uint    `gorm:"uniqueIndex"`
	Rating             float64 `uadmin:"required"`
	GrammarCorrections string  `uadmin:"html"`
	Recommendation     string  `uadmin:"html"`
	Homework           string  `uadmin:"html"`
	Remarks            string  `uadmin:"html"`
}
