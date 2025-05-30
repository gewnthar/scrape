// backend/config/config.go
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv" // For loading .env files
)

// --- Configuration Structs ---

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

type FAAURLsConfig struct {
	CdrCSV                        string
	PreferredRoutesCSV            string
	CdrEffectiveDatePage          string
	PreferredRoutesEffectiveDatePage string
	AdvisoryListPageBase          string
	// Add other URLs like RATReaderBaseURL etc. as needed
}

type LocalCSVPathsConfig struct {
	Cdr             string
	PreferredRoutes string
}

type DataFreshnessConfig struct {
	RouteDBCheckIntervalStr  string // e.g., "24h"
	FAAPublicationCycleDays int    // e.g., 56
	RouteDBCheckInterval    time.Duration
}

type ScraperSelectorsConfig struct {
	CdrEffectiveDate             string
	PreferredRoutesEffectiveDate string
}

// Config holds all application configuration.
type Config struct {
	Server           ServerConfig
	Database         DatabaseConfig
	FAAURLs          FAAURLsConfig
	LocalCSVPaths    LocalCSVPathsConfig
	DataFreshness    DataFreshnessConfig
	ScraperSelectors ScraperSelectorsConfig
}

// AppConfig is the global configuration instance.
var AppConfig Config

// --- Helper Functions for Loading ---

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("ENV: %s not set, using fallback: %s", key, fallback)
	return fallback
}

func getEnvRequired(key string) (string, error) {
	if value, exists := os.LookupEnv(key); exists {
		if value == "" {
			return "", fmt.Errorf("environment variable %s is set but empty", key)
		}
		return value, nil
	}
	return "", fmt.Errorf("environment variable %s not set and is required", key)
}

func getEnvInt(key string, fallback int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		value, err := strconv.Atoi(valueStr)
		if err == nil {
			return value
		}
		log.Printf("ENV: Could not parse %s ('%s') as int: %v. Using fallback %d.", key, valueStr, err, fallback)
	}
	log.Printf("ENV: %s not set or invalid, using fallback: %d", key, fallback)
	return fallback
}

// LoadConfig loads configuration from a specified .env file path,
// then falls back to OS environment variables.
// Pass an empty string for envFilePath to only use OS environment variables.
// Common practice: For local dev, use .env. For deployed envs, set OS env vars.
func LoadConfig(envFilePath string) error {
	if envFilePath != "" {
		// Attempt to determine absolute path if not already absolute
		if !filepath.IsAbs(envFilePath) {
			// Assuming envFilePath is relative to the current working directory.
			// If running `go run ./backend/main.go` from `scrape/`, cwd is `scrape/`.
			// If running `./scrape_app` (compiled binary) from `scrape/`, cwd is `scrape/`.
			// `godotenv.Load` will search for this path.
			// If your .env is in `scrape/.env`, and you run from `scrape/`, `envFilePath=".env"` is good.
		}
		err := godotenv.Load(envFilePath)
		if err != nil {
			// It's okay if .env is not found, might be using OS env vars.
			log.Printf("Info: Could not load .env file from '%s': %v. Relying on OS environment variables or defaults.", envFilePath, err)
		} else {
			log.Printf("Successfully loaded environment variables from %s", envFilePath)
		}
	} else {
		log.Println("Info: No .env file path specified. Relying on OS environment variables or defaults.")
	}

	// Populate ServerConfig
	AppConfig.Server.Port = getEnv("SERVER_PORT", "8080")

	// Populate DatabaseConfig - make user, password, dbname required
	var err error
	AppConfig.Database.Host = getEnv("DB_HOST", "localhost")
	AppConfig.Database.Port = getEnv("DB_PORT", "3306")
	AppConfig.Database.User, err = getEnvRequired("DB_USER")
	if err != nil { return err }
	AppConfig.Database.Password, err = getEnvRequired("DB_PASSWORD")
	if err != nil { return err }
	AppConfig.Database.DBName, err = getEnvRequired("DB_NAME")
	if err != nil { return err }


	// Populate FAAURLsConfig
	AppConfig.FAAURLs.CdrCSV = getEnv("CDR_CSV_URL", "https://www.fly.faa.gov/rmt/data_file/codedswap_db.csv")
	AppConfig.FAAURLs.PreferredRoutesCSV = getEnv("PREFERRED_ROUTES_CSV_URL", "https://www.fly.faa.gov/rmt/data_file/prefroutes_db.csv")
	AppConfig.FAAURLs.CdrEffectiveDatePage = getEnv("CDR_EFFECTIVE_DATE_PAGE_URL", "https://www.fly.faa.gov/rmt/cdm_operational_coded_departur.jsp")
	AppConfig.FAAURLs.PreferredRoutesEffectiveDatePage = getEnv("PREFERRED_ROUTES_EFFECTIVE_DATE_PAGE_URL", "https://www.fly.faa.gov/rmt/nfdc_preferred_routes_database.jsp")
	AppConfig.FAAURLs.AdvisoryListPageBase = getEnv("ADVISORY_LIST_PAGE_BASE_URL", "https://www.fly.faa.gov/adv/adv_list.jsp")

	// Populate LocalCSVPathsConfig
	// Path should be relative to where the application binary is executed.
	AppConfig.LocalCSVPaths.Cdr = getEnv("LOCAL_CDR_CSV_PATH", "./temp_data/codedswap_db.csv")
	AppConfig.LocalCSVPaths.PreferredRoutes = getEnv("LOCAL_PREFERRED_ROUTES_CSV_PATH", "./temp_data/prefroutes_db.csv")

	// Create temp_data directory if it doesn't exist
	pathsToEnsure := []string{AppConfig.LocalCSVPaths.Cdr, AppConfig.LocalCSVPaths.PreferredRoutes}
	for _, p := range pathsToEnsure {
		if p != "" {
			dir := filepath.Dir(p)
			if err := os.MkdirAll(dir, 0755); err != nil {
				// Log as warning, not fatal, as it might not be critical for all operations
				log.Printf("Warning: Failed to create directory %s: %v", dir, err)
			}
		}
	}
	
	// Populate DataFreshnessConfig
	AppConfig.DataFreshness.RouteDBCheckIntervalStr = getEnv("ROUTE_DB_CHECK_INTERVAL", "24h")
	AppConfig.DataFreshness.FAAPublicationCycleDays = getEnvInt("FAA_PUBLICATION_CYCLE_DAYS", 56)
	parsedInterval, err := time.ParseDuration(AppConfig.DataFreshness.RouteDBCheckIntervalStr)
	if err != nil {
		log.Printf("Warning: Failed to parse ROUTE_DB_CHECK_INTERVAL '%s': %v. Using default 24h.", AppConfig.DataFreshness.RouteDBCheckIntervalStr, err)
		AppConfig.DataFreshness.RouteDBCheckInterval = 24 * time.Hour
	} else {
		AppConfig.DataFreshness.RouteDBCheckInterval = parsedInterval
	}

	// Populate ScraperSelectorsConfig
	AppConfig.ScraperSelectors.CdrEffectiveDate = getEnv("SELECTOR_CDR_EFFECTIVE_DATE", "body") // QC: Update this!
	AppConfig.ScraperSelectors.PreferredRoutesEffectiveDate = getEnv("SELECTOR_PREFERRED_ROUTES_EFFECTIVE_DATE", "body") // QC: Update this!

	log.Println("Configuration loaded.")
	return nil
}