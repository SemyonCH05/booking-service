package main

import (
	"log"

	"room-booking-service/config"
	"room-booking-service/internal/app"
)

func main() {
	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	app.RunMigrations(*cfg)
	app.Run(cfg)
}
