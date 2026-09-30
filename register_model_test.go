package main

import (
	"testing"

	"github.com/uadmin/uadmin"
)

func TestRegisterModels(t *testing.T) {
	registerModels()

	models := []string{
		"assessment",
		"class",
		"course",
		"coursematerial",
		"enrollment",
		"homework",
		"invoice",
		"level",
		"material",
		"package",
		"student",
		"teacher",
	}

	for _, modelName := range models {
		t.Run(modelName, func(t *testing.T) {
			if _, ok := uadmin.NewModel(modelName, false); !ok {
				t.Fatalf("expected %s to be registered", modelName)
			}
		})
	}
}
