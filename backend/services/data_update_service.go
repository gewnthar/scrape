// backend/services/data_update_service.go
package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gewnthar/scrape/backend/config" // Adjust to your module path
	"github.com/gewnthar/scrape/backend/database"
	"github.com/gewnthar/scrape/backend/models"
	"github.com/gewnthar/scrape/backend/scraper"
)

// Simulating a simple in-memory store or a place to get this from DB later
var lastKnownEffectiveDates = make(map[string]time.Time) // Key: "CDR" or "PreferredRoutes", Value: EffectiveUntil

// UpdateStaticRouteDataIfNeeded checks if an update for a static route data source is needed
// by comparing the "Effective Until" date on the FAA website with the last known one.
// sourceName should be "CDR" or "PreferredRoutes".
// cssSelectorForDate is the specific CSS selector for the effective date string on the FAA page.
func UpdateStaticRouteDataIfNeeded(sourceName string, cssSelectorForDate string) error {
	log.Printf("DataUpdateService: Checking if update is needed for %s data...\n", sourceName)

	var currentFAAEffectiveInfo *models.DataSourceEffectiveInfo
	var err error

	// Step 1: Get current effective dates from FAA website
	switch sourceName {
	case "CDR":
		currentFAAEffectiveInfo, err = scraper.ScrapeEffectiveDatesForCDR(cssSelectorForDate)
	case "PreferredRoutes":
		currentFAAEffectiveInfo, err = scraper.ScrapeEffectiveDatesForPreferredRoutes(cssSelectorForDate)
	default:
		return fmt.Errorf("unknown data source name: %s", sourceName)
	}

	if err != nil {
		return fmt.Errorf("failed to scrape effective dates for %s: %w", sourceName, err)
	}
	if currentFAAEffectiveInfo == nil {
		return fmt.Errorf("no effective date info retrieved for %s", sourceName)
	}

	log.Printf("DataUpdateService: Current FAA effective until date for %s: %s\n", sourceName, currentFAAEffectiveInfo.EffectiveUntil.Format("2006-01-02"))

	// Step 2: Compare with last known effective date (simple in-memory version for now)
	// In a real system, you might get this from a database table that logs data versions.
	lastProcessedUntil, found := lastKnownEffectiveDates[sourceName]
	updateNeeded := false

	if !found {
		log.Printf("DataUpdateService: No previous effective date found for %s. Update needed.\n", sourceName)
		updateNeeded = true
	} else if currentFAAEffectiveInfo.EffectiveUntil.After(lastProcessedUntil) {
		log.Printf("DataUpdateService: FAA effective date for %s (%s) is newer than last processed (%s). Update needed.\n",
			sourceName,
			currentFAAEffectiveInfo.EffectiveUntil.Format("2006-01-02"),
			lastProcessedUntil.Format("2006-01-02"))
		updateNeeded = true
	} else {
		log.Printf("DataUpdateService: Current %s data (effective until %s) is up-to-date.\n",
			sourceName, lastProcessedUntil.Format("2006-01-02"))
	}
	
	// Incorporate 56-day cycle prediction (optional enhancement here or in scheduler)
	// For example, if !updateNeeded but today is near/past lastProcessedUntil + 56 days,
	// you might force a check or log a warning.

	if updateNeeded {
		log.Printf("DataUpdateService: Proceeding with update for %s...\n", sourceName)
		// Pass the effective dates scraped from FAA to the ForceUpdate function
		// so the new data is tagged with its correct validity period.
		return ForceUpdateStaticRouteData(sourceName, currentFAAEffectiveInfo)
	}

	return nil
}

// ForceUpdateStaticRouteData forces a download, parse, and save of a static route data source.
// If provided, `knownEffectiveInfo` will be used to tag the data; otherwise, it will be scraped.
func ForceUpdateStaticRouteData(sourceName string, knownEffectiveInfo *models.DataSourceEffectiveInfo) error {
	log.Printf("DataUpdateService: Forcing update for %s data...\n", sourceName)

	var localPath string
	var err error
	currentEffectiveInfo := knownEffectiveInfo

	// If effective info isn't passed in (e.g. direct manual refresh without prior check), scrape it.
	if currentEffectiveInfo == nil {
		log.Printf("DataUpdateService: No known effective info for %s, scraping live dates...\n", sourceName)
		// QC Note: The CSS selectors will be passed from a handler or config for these.
		// For now, using empty string will make scraper use its default (which needs fixing by QC).
		cssSelector := "" // Placeholder - should be passed or configured
		switch sourceName {
		case "CDR":
			currentEffectiveInfo, err = scraper.ScrapeEffectiveDatesForCDR(cssSelector)
		case "PreferredRoutes":
			currentEffectiveInfo, err = scraper.ScrapeEffectiveDatesForPreferredRoutes(cssSelector)
		default:
			return fmt.Errorf("unknown data source name for force update: %s", sourceName)
		}
		if err != nil {
			return fmt.Errorf("failed to scrape effective dates during force update for %s: %w", sourceName, err)
		}
		if currentEffectiveInfo == nil {
			return fmt.Errorf("could not retrieve effective date info for %s during force update", sourceName)
		}
		log.Printf("DataUpdateService: Scraped live FAA effective dates for %s: From %s, Until %s\n",
			sourceName,
			currentEffectiveInfo.EffectiveFrom.Format("2006-01-02"),
			currentEffectiveInfo.EffectiveUntil.Format("2006-01-02"))
	}


	// Step 1: Download CSV
	switch sourceName {
	case "CDR":
		localPath, err = scraper.DownloadCdrCsv()
	case "PreferredRoutes":
		localPath, err = scraper.DownloadPreferredRoutesCsv()
	default:
		return fmt.Errorf("unknown data source name for download: %s", sourceName)
	}

	if err != nil {
		return fmt.Errorf("failed to download %s CSV: %w", sourceName, err)
	}
	defer os.Remove(localPath) // Clean up downloaded file after processing

	// Step 2: Open and Parse CSV
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded file %s: %w", localPath, err)
	}
	defer file.Close()

	sourceFileName := filepath.Base(localPath) // e.g., "codedswap_db.csv"
	// We can enhance sourceFileName with a date if needed:
	// sourceFileName = fmt.Sprintf("%s_%s", filepath.Base(localPath), time.Now().Format("20060102"))


	// Step 3: Save to Database
	switch sourceName {
	case "CDR":
		cdrRoutes, parseErr := scraper.ParseCdrCsv(file)
		if parseErr != nil {
			return fmt.Errorf("failed to parse CDR CSV from %s: %w", localPath, parseErr)
		}
		dbErr := database.SaveCdrRoutes(cdrRoutes, sourceFileName, &currentEffectiveInfo.EffectiveFrom, &currentEffectiveInfo.EffectiveUntil)
		if dbErr != nil {
			return fmt.Errorf("failed to save CDR routes to database: %w", dbErr)
		}
	case "PreferredRoutes":
		prefRoutes, parseErr := scraper.ParsePreferredRoutesCsv(file)
		if parseErr != nil {
			return fmt.Errorf("failed to parse Preferred Routes CSV from %s: %w", localPath, parseErr)
		}
		dbErr := database.SavePreferredRoutes(prefRoutes, sourceFileName, &currentEffectiveInfo.EffectiveFrom, &currentEffectiveInfo.EffectiveUntil)
		if dbErr != nil {
			return fmt.Errorf("failed to save Preferred Routes to database: %w", dbErr)
		}
	default:
		return fmt.Errorf("unknown data source name for saving: %s", sourceName)
	}

	// Update our simple in-memory store of last known effective dates
	lastKnownEffectiveDates[sourceName] = currentEffectiveInfo.EffectiveUntil
	log.Printf("DataUpdateService: Successfully updated %s data. New effective until date: %s\n",
		sourceName, currentEffectiveInfo.EffectiveUntil.Format("2006-01-02"))

	return nil
}

// InitLastKnownEffectiveDates could be called at startup to populate 'lastKnownEffectiveDates'
// by querying the MAX(effective_date_end) from your cdm_routes and nfdc_preferred_routes tables
// for each source type. This makes the "UpdateIfNeeded" logic more robust across restarts.
// For now, it's a simple in-memory map that resets on restart.
func InitLastKnownEffectiveDates() {
    // Placeholder: In a real app, query DB for max effective_date_end for each source
    // For example:
    // SELECT MAX(effective_date_end) FROM cdm_routes WHERE source_file LIKE 'codedswap_db.csv%';
    // SELECT MAX(effective_date_end) FROM nfdc_preferred_routes WHERE source_file LIKE 'prefroutes_db.csv%';
    // And then populate lastKnownEffectiveDates map.
    log.Println("DataUpdateService: LastKnownEffectiveDates initialized (currently in-memory, resets on start).")
}