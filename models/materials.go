package models

import "github.com/uadmin/uadmin"

type Material struct {
	uadmin.Model
	Name   string `uadmin:"required"`
	File   string `uadmin:"required"`
	Active bool   `uadmin:"required"`
}

// TODO: Create a helper function to upload the files to the file bucket
