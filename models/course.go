package models

import "github.com/uadmin/uadmin"

type Course struct {
	uadmin.Model
	Title       string `uadmin:"required"`
	Description string `uadmin:"list_exclude"`
	Level       Level
	LevelID     uint
	Active      bool
}

func (c Course) String() string {
	return c.Title
}

func (c *Course) Save() map[string]any {
	uadmin.Save(c)
	return map[string]any{
		"id": c.ID,
	}
}
