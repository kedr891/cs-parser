package main

import (
	"context"
	"log"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/app"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	if err := app.RunNotification(context.Background(), cfg); err != nil {
		log.Fatalf("app terminated: %v", err)
	}
}
