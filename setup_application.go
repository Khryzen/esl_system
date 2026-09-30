package main

import (
	"github.com/uadmin/uadmin"
)

func setupApplication(config AppConfig) {
	if config.Database != nil {
		uadmin.Database = config.Database
	}

	registerModels()
	initializeData()
	registerRoutes()
	configureServer(config)
}
