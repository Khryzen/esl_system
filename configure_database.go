package main

import "github.com/uadmin/uadmin"

func configureDatabase(config AppConfig) {
	if config.Database != nil {
		uadmin.Database = config.Database
	}
}
