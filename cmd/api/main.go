package main

import (
	"log"

	"go-be-mono-commerce/internal/config"
	"go-be-mono-commerce/internal/server"
)

func main() {
	cfg := config.Load()
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}
	if err := srv.Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
