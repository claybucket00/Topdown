package main

import (
	"flag"
	"log"
	"topdown/internal/api"
)

func main() {
	port := flag.String("port", "8080", "Port to run the API server on")
	demoPath := flag.String("demos", "./demos", "Path to store parsed demos")
	workers := flag.Int("workers", 1, "Number of parallel workers for parsing")
	flag.Parse()

	// Create API server
	server := api.NewServer(*demoPath, *workers)

	// Start server
	address := ":" + *port
	log.Printf("Starting API server on %s\n", address)
	if err := server.Start(address); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
