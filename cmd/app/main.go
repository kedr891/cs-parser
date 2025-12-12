package main

import (
	"log"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/app"
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
