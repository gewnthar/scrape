// backend/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gewnthar/scrape/backend/config"
	"github.com/gewnthar/scrape/backend/database"
	"github.com/gewnthar/scrape/backend/handlers" // IMPORT handlers package
	"github.com/gewnthar/scrape/backend/services" // IMPORT services for InitLastKnownEffectiveDates
)

func main() {
	log.Println("Starting FAA DST Backend Application...")

	configPath := "backend/config/config.yaml" 
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "config/config.yaml" 
		if _, errFallback := os.Stat(configPath); os.IsNotExist(errFallback) {
			log.Fatalf("Config file not found at default paths. Error: %v", errFallback) // Use errFallback
		}
	}

	err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	log.Printf("Configuration loaded. Server port: %s, DB name: %s",
		config.AppConfig.Server.Port, config.AppConfig.Database.DBName)
	log.Printf("CDR CSV URL: %s", config.AppConfig.FAAURLs.CdrCSV) // Example of accessing config
	log.Printf("Preferred Routes Date Selector: %s", config.AppConfig.ScraperSelectors.PreferredRoutesEffectiveDate)


	err = database.InitDB(config.AppConfig.Database)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer database.CloseDB()

	// Initialize any necessary service states (like last known effective dates from DB)
	services.InitLastKnownEffectiveDates() // Call the init function

	// --- Setup HTTP routes ---
	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := database.DB.Ping(); err != nil {
			http.Error(w, `{"status": "error", "message": "database connection error"}`, http.StatusInternalServerError)
			log.Printf("Health check failed: DB ping error: %v", err)
			return
		}
		fmt.Fprintln(w, `{"status": "ok", "message": "FAA DST backend is healthy"}`)
	})

	// Admin routes for managing static data
	http.HandleFunc("/api/admin/refresh-routes/", handlers.ForceRefreshStaticRouteDataHandler)     // Path ends with / to catch sub-paths
	http.HandleFunc("/api/admin/check-update-routes/", handlers.CheckAndUpdateStaticRouteDataHandler) // Path ends with /

	// Add other handlers for advisory_handler.go and route_handler.go here later
	// http.HandleFunc("/api/advisories", handlers.GetAdvisoriesHandler) 
	// http.HandleFunc("/api/routes/find", handlers.FindRoutesHandler)


	serverAddr := ":" + config.AppConfig.Server.Port
	log.Printf("Server starting on http://localhost%s\n", serverAddr)
	err = http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}