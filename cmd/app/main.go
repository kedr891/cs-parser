package main

import (
	"log"

	"github.com/cs-parser/config"
	"github.com/cs-parser/internal/app"
)

func main() {
	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	app.Run(cfg)
}
