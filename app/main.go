package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"net/http"
)

func main() {
	cfg := LoadConfig()

	instances, cancel, err := InitializeInstances(cfg)
	if err != nil {
		log.Fatalf("Failed initializing instances: %v", err)
	}

	server := NewServer(instances)
	go func() {
		log.Printf("MinecraftStats Go Server listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Received termination signal, shutting down...")

	cancel() // stop instance loops

	ctx, cancelHTTP := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelHTTP()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}

	log.Println("Shutdown complete")
}