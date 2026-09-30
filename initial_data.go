package main

import (
	"github.com/Khryzen/esl_system/models"
	"github.com/uadmin/uadmin"
)

func initializeData() {
	levelData := []models.Level{
		{
			Level: "Newbie",
		},
		{
			Level: "Beginner",
		},
		{
			Level: "Intermediate",
		},
		{
			Level: "Moderate",
		},
		{
			Level: "Proficient",
		},
	}

	level := models.Level{}
	if uadmin.Count(&level, "id > 0") == 0 {
		for i := range levelData {
			uadmin.Save(&levelData[i])
		}
	}
}
