package main

import (
	"log"
	"os"

	"spawnr/internal/handlers"
	"spawnr/internal/k8s"
	"spawnr/internal/server"
)

func main() {
	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create handlers
	h := handlers.New(k8sClient)

	// Create server
	srv := server.New(h)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Starting server on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
