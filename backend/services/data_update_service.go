// backend/services/data_update_service.go
package services

import (
	// fmt, log, os, path/filepath, time, net/url (for InitLastKnownEffectiveDates)
	// config, database, models, scraper (as before)
	"fmt"
	"io" // Needed for parseFunc signature
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gewnthar/scrape/backend/config"
	"github.com/gewnthar/scrape/backend/database"
	"github.com/gewnthar/scrape/backend/models"
	"github.com/gewnthar/scrape/backend/scraper"
)

var lastKnownEffectiveDates = make(map[string]time.Time)

const (
	sourceCDR             = "CDR"
	sourcePreferredRoutes = "PreferredRoutes"
)

func InitLastKnownEffectiveDates() {
	log.Println("Service: Initializing last known effective dates for static route data...")
	// For CDRs
	var cdrPattern string
	if u, err := url.Parse(config.AppConfig.FAAURLs.CdrCSV); err == nil {
		cdrPattern = filepath.Base(u.Path)
	} else {
		cdrPattern = "codedswap_db.csv" // Fallback
		log.Printf("WARN Service: Could not parse CdrCSV URL, using default pattern '%s'", cdrPattern)
	}
	cdrDate, err := database.GetMaxEffectiveEndDateForSource(cdrPattern) // Pass base filename pattern
	if err != nil {
		log.Printf("ERROR Service: Failed to get max effective end date for CDRs from DB: %v\n", err)
	} else if cdrDate != nil {
		lastKnownEffectiveDates[sourceCDR] = *cdrDate
		log.Printf("INFO Service: Initialized last known CDR effective until date from DB: %s\n", cdrDate.Format("2006-01-02"))
	} else {
		log.Println("INFO Service: No existing CDR effective end date found in DB.")
	}

	// For Preferred Routes
	var prefPattern string
	if u, err := url.Parse(config.AppConfig.FAAURLs.PreferredRoutesCSV); err == nil {
		prefPattern = filepath.Base(u.Path)
	} else {
		prefPattern = "prefroutes_db.csv" // Fallback
		log.Printf("WARN Service: Could not parse PreferredRoutesCSV URL, using default pattern '%s'", prefPattern)
	}
	prefDate, err := database.GetMaxEffectiveEndDateForSource(prefPattern) // Pass base filename pattern
	if err != nil {
		log.Printf("ERROR Service: Failed to get max effective end date for Preferred Routes from DB: %v\n", err)
	} else if prefDate != nil {
		lastKnownEffectiveDates[sourcePreferredRoutes] = *prefDate
		log.Printf("INFO Service: Initialized last known Preferred Routes effective until date from DB: %s\n", prefDate.Format("2006-01-02"))
	} else {
		log.Println("INFO Service: No existing Preferred Routes effective end date found in DB.")
	}
	// TODO: Enhance this to use the `data_source_versions` table for more robust tracking.
}

func UpdateStaticRouteDataIfNeeded(sourceName string, cssSelectorForDate string) error {
	log.Printf("Service: Checking if update is needed for %s data (selector: '%s')...\n", sourceName, cssSelectorForDate)
	log.Println("Service: NOTE - 'UpdateIfNeeded' functionality is currently limited as live 'Effective Date' scraping is deferred pending CSS selector finalization.")

	var currentFAAEffectiveInfo *models.DataSourceEffectiveInfo
	var err error

	// Step 1: Get current effective dates from FAA website (THIS PART IS DEFERRED UNTIL SELECTORS ARE FINAL)
	// For now, this will likely fail or use placeholder selectors if called.
	// We can simulate it not finding a newer date to prevent updates unless forced.
	log.Printf("Service: Skipping live FAA effective date check for %s as per current focus. To enable, provide correct CSS selectors.", sourceName)
	// To simulate no update needed based on date check:
	// return nil

	// If we were to proceed with the check:

	switch sourceName {
	case sourceCDR:
		currentFAAEffectiveInfo, err = scraper.ScrapeEffectiveDatesForCDR(cssSelectorForDate)
	case sourcePreferredRoutes:
		currentFAAEffectiveInfo, err = scraper.ScrapeEffectiveDatesForPreferredRoutes(cssSelectorForDate)
	default:
		return fmt.Errorf("unknown data source name: %s", sourceName)
	}

	if err != nil {
		return fmt.Errorf("failed to scrape effective dates for %s: %w. QC: Verify CSS selector '%s'", sourceName, err, cssSelectorForDate)
	}
	if currentFAAEffectiveInfo == nil {
		return fmt.Errorf("no effective date info retrieved from FAA for %s. QC: Verify CSS selector '%s'", sourceName, cssSelectorForDate)
	}
	log.Printf("Service: Current FAA effective until date for %s: %s\n", sourceName, currentFAAEffectiveInfo.EffectiveUntil.Format("2006-01-02"))

	lastProcessedUntil, found := lastKnownEffectiveDates[sourceName]
	updateNeeded := false
	currentFAAUntilDate := time.Date(currentFAAEffectiveInfo.EffectiveUntil.Year(), currentFAAEffectiveInfo.EffectiveUntil.Month(), currentFAAEffectiveInfo.EffectiveUntil.Day(), 0, 0, 0, 0, time.UTC)

	if !found {
		updateNeeded = true
	} else {
		lastProcessedUntilDate := time.Date(lastProcessedUntil.Year(), lastProcessedUntil.Month(), lastProcessedUntil.Day(), 0, 0, 0, 0, time.UTC)
		if currentFAAUntilDate.After(lastProcessedUntilDate) {
			updateNeeded = true
		}
	}

	if updateNeeded {
		log.Printf("Service: Update detected as needed for %s based on FAA effective dates.\n", sourceName)
		return ForceUpdateStaticRouteData(sourceName, currentFAAEffectiveInfo) // Pass the live info
	} else {
		log.Printf("Service: No update deemed necessary for %s based on FAA effective dates.\n", sourceName)
	}

	log.Printf("Service: 'UpdateIfNeeded' for %s concluded (live date check deferred).\n", sourceName)
	return nil
}

// ForceUpdateStaticRouteData forces a download, parse, and save of a static route data source.
// If `liveEffectiveInfoFromFAA` is nil (e.g. for a purely manual refresh bypassing date check),
// then effective dates stored in DB for the new data will be NULL.
func ForceUpdateStaticRouteData(sourceName string, liveEffectiveInfoFromFAA *models.DataSourceEffectiveInfo) error {
	log.Printf("Service: Forcing update for %s data...\n", sourceName)

	var localPath string
	var csvURL string // Not strictly needed here as downloadFunc gets from config
	var downloadFunc func() (string, error)
	var parseFunc func(io.Reader) (interface{}, error)
	var saveFunc func(interface{}, string, *time.Time, *time.Time) error

	var effectiveFrom, effectiveUntil *time.Time
	var sourceFileForDB string

	if liveEffectiveInfoFromFAA != nil {
		effectiveFrom = &liveEffectiveInfoFromFAA.EffectiveFrom
		effectiveUntil = &liveEffectiveInfoFromFAA.EffectiveUntil
		log.Printf("Service: Using provided effective dates for %s: From %s, Until %s\n",
			sourceName, effectiveFrom.Format("2006-01-02"), effectiveUntil.Format("2006-01-02"))
	} else {
		log.Printf("Service: No live effective date info provided for %s. Effective dates in DB will be NULL for this batch.", sourceName)
		// effectiveFrom and effectiveUntil remain nil
	}

	// Configure download, parse, and save functions based on sourceName
	switch sourceName {
	case sourceCDR:
		csvURL = config.AppConfig.FAAURLs.CdrCSV // For logging
		downloadFunc = scraper.DownloadCdrCsv
		parseFunc = func(r io.Reader) (interface{}, error) { return scraper.ParseCdrCsv(r) }
		saveFunc = func(data interface{}, sf string, es *time.Time, ee *time.Time) error {
			return database.SaveCdrRoutes(data.([]models.CdrRoute), sf, es, ee)
		}
	case sourcePreferredRoutes:
		csvURL = config.AppConfig.FAAURLs.PreferredRoutesCSV // For logging
		downloadFunc = scraper.DownloadPreferredRoutesCsv
		parseFunc = func(r io.Reader) (interface{}, error) { return scraper.ParsePreferredRoutesCsv(r) }
		saveFunc = func(data interface{}, sf string, es *time.Time, ee *time.Time) error {
			return database.SavePreferredRoutes(data.([]models.PreferredRoute), sf, es, ee)
		}
	default:
		return fmt.Errorf("unknown data source name for forced update: %s", sourceName)
	}

	// Step 1: Download CSV
	log.Printf("Service: Downloading %s CSV from %s\n", sourceName, csvURL) // Log the URL being used
	localPath, err := downloadFunc()
	if err != nil {
		return fmt.Errorf("failed to download %s CSV: %w", sourceName, err)
	}
	log.Printf("Service: Downloaded %s to %s\n", sourceName, localPath)
	defer func() {
		log.Printf("Service: Cleaning up temporary file: %s\n", localPath)
		if err := os.Remove(localPath); err != nil {
			log.Printf("ERROR Service: Failed to remove temporary file %s: %v\n", localPath, err)
		}
	}()

	// Step 2: Open and Parse CSV
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded file %s: %w", localPath, err)
	}
	defer file.Close()

	parsedData, err := parseFunc(file)
	if err != nil {
		return fmt.Errorf("failed to parse %s CSV from %s: %w", sourceName, localPath, err)
	}

	// Step 3: Save to Database
	sourceFileForDB = filepath.Base(localPath)
	if effectiveFrom != nil { // Append effective_from date to source file name if known
		sourceFileForDB = fmt.Sprintf("%s_%s", filepath.Base(localPath), effectiveFrom.Format("20060102"))
	}
	
	err = saveFunc(parsedData, sourceFileForDB, effectiveFrom, effectiveUntil)
	if err != nil {
		return fmt.Errorf("failed to save %s routes to database (source file: %s): %w", sourceName, sourceFileForDB, err)
	}

	// Step 4: Update our simple in-memory store (and ideally persistent store)
	if effectiveUntil != nil {
		lastKnownEffectiveDates[sourceName] = *effectiveUntil
		log.Printf("Service: Successfully forced update for %s data. New effective until date in memory: %s\n",
			sourceName, effectiveUntil.Format("2006-01-02"))
		// TODO: Persist this update to data_source_versions table
		// e.g., database.LogDataSourceVersionUpdate(sourceName, csvURL, sourceFileForDB, effectiveFrom, effectiveUntil, time.Now())
	} else {
		log.Printf("Service: Successfully forced update for %s data. Effective dates are NULL for this batch.\n", sourceName)
		// If effectiveUntil is nil, we might want to remove it from lastKnownEffectiveDates
		// or have a strategy for when to re-check if dates were NULL.
		// For now, if it was nil, the map entry for sourceName won't be updated with a specific date.
	}

	return nil
}