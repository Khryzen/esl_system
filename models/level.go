package models

import "github.com/uadmin/uadmin"

type Level struct {
	uadmin.Model
	Level string `uadmin:"required"`
}

func (l Level) String() string {
	return l.Level
}
