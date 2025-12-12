package main

import (
	"astra_core/api"
	"astra_core/database"
	"astra_core/monitor"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	log.Println("Starting Astra Core...")

	dbPath := getEnv("DB_PATH", "astranet.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		log.Println("WARNING: DISCORD_WEBHOOK_URL is not set. Notifications will be disabled.")
	}

	appIDStr := getEnv("APP_ID", "730")
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		log.Fatalf("Invalid APP_ID: %v", err)
	}

	apiPort := getEnv("API_PORT", "8080")

	mon := monitor.NewMonitor(appID, db)

	apiServer := api.NewServer(mon)
	go apiServer.Start(":" + apiPort)

	go mon.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}
