package main

import "github.com/uadmin/uadmin"

func configureServer(config AppConfig) {
	uadmin.RootURL = "/admin/"
	uadmin.Port = config.Port
}
