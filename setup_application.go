package main

func setupApplication(config AppConfig) {
	configureDatabase(config)
	registerModels()
	registerRoutes()
	configureServer(config)
}
