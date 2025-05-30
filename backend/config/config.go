// backend/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port string `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type FAAURLsConfig struct {
	CdrCSV             string `yaml:"cdr_csv"`
	PreferredRoutesCSV string `yaml:"preferred_routes_csv"`
	// Add other URLs here as needed
}

type LocalCSVPathsConfig struct {
	Cdr             string `yaml:"cdr"`
	PreferredRoutes string `yaml:"preferred_routes"`
}

type DataFreshnessConfig struct {
	RouteDBCheckIntervalStr  string `yaml:"route_db_check_interval"`
	FAAPublicationCycleDays int    `yaml:"faa_publication_cycle_days"`
	RouteDBCheckInterval    time.Duration // Parsed duration
}

type ScraperSelectorsConfig struct {
	CdrEffectiveDate             string `yaml:"cdr_effective_date"`
	PreferredRoutesEffectiveDate string `yaml:"preferred_routes_effective_date"`
}

type Config struct {
	Server           ServerConfig           `yaml:"server"`
	Database         DatabaseConfig         `yaml:"database"`
	FAAURLs          FAAURLsConfig          `yaml:"faa_urls"`
	LocalCSVPaths    LocalCSVPathsConfig    `yaml:"local_csv_paths"`
	DataFreshness    DataFreshnessConfig    `yaml:"data_freshness"`
	ScraperSelectors ScraperSelectorsConfig `yaml:"scraper_selectors"` // Add this line
}

var AppConfig Config

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(configPath string) error {
	// Default path if not provided
	if configPath == "" {
		// Assuming config.yaml is in the same directory as the executable or one level up in a 'config' folder
		// This might need adjustment based on your deployment structure
		absPath, err := filepath.Abs(".")
		if err != nil {
			return fmt.Errorf("error getting absolute path: %w", err)
		}
		// Try backend/config.yaml, ./config.yaml, then ./config/config.yaml
		// This logic can be simplified if you always run from a specific directory
		// or always provide the path.
		// For now, let's assume it's in backend/config/ relative to project root,
		// or just 'config.yaml' if running from backend/
		
		// A common pattern: run 'backend' executable from 'scrape/' root.
		// So config would be at 'backend/config/config.yaml'
		// If running from 'backend/', then 'config/config.yaml'
		// For simplicity now, let's assume configPath is passed correctly
		// or it's just "config.yaml" if running from backend/config/ (which is not typical)
		// Let's adjust to be relative to the 'backend' directory if path is empty
		// and executable is in 'backend'
		// path, _ := os.Getwd()
		// fmt.Println("Current working directory:", path)
		// A robust way is to expect it relative to where the binary is run,
		// or use an environment variable for the config path.
		// Let's try to find it in common locations relative to current dir for now.
		potentialPaths := []string{
			"config.yaml",          // If running from backend/config/
			"../config/config.yaml", // If running from backend/ (binary in backend/)
			"./backend/config/config.yaml", // If running from project root (scrape/)
		}

		foundPath := ""
		for _, p := range potentialPaths {
			if _, err := os.Stat(p); err == nil {
				configPath = p
				foundPath = p
				break
			}
		}
		if foundPath == "" {
			return fmt.Errorf("config.yaml not found in standard locations")
		}
		fmt.Printf("Loading configuration from: %s\n", foundPath)
	}


	file, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	err = yaml.Unmarshal(file, &AppConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Parse durations
	if AppConfig.DataFreshness.RouteDBCheckIntervalStr != "" {
		AppConfig.DataFreshness.RouteDBCheckInterval, err = time.ParseDuration(AppConfig.DataFreshness.RouteDBCheckIntervalStr)
		if err != nil {
			return fmt.Errorf("failed to parse RouteDBCheckInterval: %w", err)
		}
	} else {
		AppConfig.DataFreshness.RouteDBCheckInterval = 24 * time.Hour // Default
	}


	// Create temp_data directory if it doesn't exist for local CSVs
	// This assumes LocalCSVPaths.Cdr or .PreferredRoutes might include a directory like "./temp_data/"
    if AppConfig.LocalCSVPaths.Cdr != "" {
		if err := os.MkdirAll(filepath.Dir(AppConfig.LocalCSVPaths.Cdr), 0755); err != nil {
			return fmt.Errorf("failed to create directory for CDR CSV: %w", err)
		}
	}
	if AppConfig.LocalCSVPaths.PreferredRoutes != "" {
		if err := os.MkdirAll(filepath.Dir(AppConfig.LocalCSVPaths.PreferredRoutes), 0755); err != nil {
			return fmt.Errorf("failed to create directory for Preferred Routes CSV: %w", err)
		}
	}


	return nil
}