// backend/scraper/csv_downloader.go
package scraper

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gewnthar/scrape/backend/config" // Adjust to your module path
)

// DownloadFile is a utility function to download a file from a URL and save it to a local path.
// It returns an error if any step fails.
func DownloadFile(url string, localSavePath string) error {
	log.Printf("Attempting to download file from URL: %s to local path: %s\n", url, localSavePath)

	// Create a new HTTP client with a timeout
	client := http.Client{
		Timeout: 30 * time.Second, // Sensible timeout for a file download
	}

	// Make the GET request
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to make GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file from %s: received status code %d", url, resp.StatusCode)
	}

	// Ensure the directory for the local save path exists
	dir := filepath.Dir(localSavePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create the local file
	outFile, err := os.Create(localSavePath)
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", localSavePath, err)
	}
	defer outFile.Close()

	// Copy the response body to the local file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy downloaded content to %s: %w", localSavePath, err)
	}

	log.Printf("Successfully downloaded %s to %s\n", url, localSavePath)
	return nil
}

// DownloadCdrCsv downloads the CDR CSV file from the URL specified in the config
// and saves it to the local path specified in the config.
// It returns the local path of the downloaded file or an error.
func DownloadCdrCsv() (string, error) {
	cdrURL := config.AppConfig.FAAURLs.CdrCSV
	localPath := config.AppConfig.LocalCSVPaths.Cdr

	if cdrURL == "" {
		return "", fmt.Errorf("CDR CSV URL is not configured")
	}
	if localPath == "" {
		return "", fmt.Errorf("local save path for CDR CSV is not configured")
	}

	err := DownloadFile(cdrURL, localPath)
	if err != nil {
		return "", fmt.Errorf("failed to download CDR CSV: %w", err)
	}
	return localPath, nil
}

// DownloadPreferredRoutesCsv downloads the Preferred Routes CSV file from the URL specified in the config
// and saves it to the local path specified in the config.
// It returns the local path of the downloaded file or an error.
func DownloadPreferredRoutesCsv() (string, error) {
	prefRoutesURL := config.AppConfig.FAAURLs.PreferredRoutesCSV
	localPath := config.AppConfig.LocalCSVPaths.PreferredRoutes

	if prefRoutesURL == "" {
		return "", fmt.Errorf("Preferred Routes CSV URL is not configured")
	}
	if localPath == "" {
		return "", fmt.Errorf("local save path for Preferred Routes CSV is not configured")
	}

	err := DownloadFile(prefRoutesURL, localPath)
	if err != nil {
		return "", fmt.Errorf("failed to download Preferred Routes CSV: %w", err)
	}
	return localPath, nil
}