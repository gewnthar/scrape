// backend/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gewnthar/scrape/backend/config"  // Adjust to your module path
	"github.com/gewnthar/scrape/backend/database"
	"github.com/gewnthar/scrape/backend/handlers" // For admin_handler, advisory_handler (later), route_handler
	"github.com/gewnthar/scrape/backend/services" // For InitLastKnownEffectiveDates
)

func main() {
	// Setup structured logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("INFO: Starting FAA DST Backend Application...")

	// --- Configuration ---
	// Expect .env file to be in the project root, relative to where the binary is typically run.
	// If running `go run ./backend/main.go` from project root `scrape/`, envPath should be ".env".
	// If `scrape_app` binary is in `scrape/` and run from there, envPath is ".env".
	envPath := ".env" 
	err := config.LoadConfig(envPath)
	if err != nil {
		log.Fatalf("CRITICAL: Error loading configuration: %v. Ensure .env file is at '%s' or environment variables are set.", err, envPath)
	}
	log.Printf("INFO: Configuration loaded. Server port: %s, DB: %s@%s/%s",
		config.AppConfig.Server.Port,
		config.AppConfig.Database.User,
		config.AppConfig.Database.Host,
		config.AppConfig.Database.DBName)

	// --- Database ---
	if err := database.InitDB(config.AppConfig.Database); err != nil {
		log.Fatalf("CRITICAL: Error initializing database: %v", err)
	}
	defer database.CloseDB()

	// --- Services Initialization ---
	// (Example: populate in-memory cache of last known effective dates from DB at startup)
	services.InitLastKnownEffectiveDates() // Implemented in services/data_update_service.go

	// --- HTTP Router Setup ---
	mux := http.NewServeMux() // Using the standard library's ServeMux

	// Health check
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if dbErr := database.DB.Ping(); dbErr != nil {
			// Use the helper from one of your handler packages or define a local one
			// For now, simple http.Error and log
			log.Printf("ERROR: Health check failed: DB ping error: %v", dbErr)
			http.Error(w, `{"status":"error", "message":"database connection error"}`, http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, `{"status":"ok", "message":"FAA DST backend is healthy"}`)
		log.Println("INFO: Health check successful")
	})

	// Admin routes for managing static data (from handlers/admin_handler.go)
	// The trailing slash is important for net/http's ServeMux to match prefix
	mux.HandleFunc("/api/admin/refresh-routes/", handlers.ForceRefreshStaticRouteDataHandler)
	mux.HandleFunc("/api/admin/check-update-routes/", handlers.CheckAndUpdateStaticRouteDataHandler)
	
	// Advisory Handlers (from handlers/advisory_handler.go)
	mux.HandleFunc("/api/advisories/exploratory-search", handlers.ExploratoryAdvisorySearchHandler)
	mux.HandleFunc("/api/advisories", handlers.GetAdvisorySummariesHandler) 
	mux.HandleFunc("/api/advisories/detail", handlers.GetAdvisoryDetailHandler) 
	mux.HandleFunc("/api/advisories/detail/save", handlers.ConfirmSaveAdvisoryDetailHandler)

	// Route Finding Handler (from handlers/route_handler.go)
	mux.HandleFunc("/api/routes/find", handlers.FindRoutesHandler) 

	// --- HTTP Server Start with Graceful Shutdown ---
	serverAddr := ":" + config.AppConfig.Server.Port
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: mux, 
		ReadTimeout:  15 * time.Second, // Increased slightly
		WriteTimeout: 15 * time.Second, // Increased slightly
		IdleTimeout:  30 * time.Second, // Increased slightly
	}

	// Channel to listen for interrupt or terminate signals from the OS.
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Goroutine to start the server
	go func() {
		log.Printf("INFO: Server starting on http://localhost%s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("FATAL: Could not listen on %s: %v\n", serverAddr, err)
		}
	}()
	log.Println("INFO: Server startup process initiated.")

	// Block until a signal is received.
	<-done
	log.Println("INFO: Server received shutdown signal. Attempting graceful shutdown...")

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("FATAL: Server graceful shutdown failed: %v", err)
	}
	log.Println("INFO: Server gracefully stopped.")
}