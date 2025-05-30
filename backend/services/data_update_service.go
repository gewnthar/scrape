// backend/services/data_update_service.go
package services

import (
	"fmt"
	"io" 
	"log"
	"os"
	"path/filepath"
	// "strings" // Not strictly needed in this version of InitLastKnownEffectiveDates
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
	// Call GetMaxEffectiveEndDateForSource with the generic identifier "CDR"
	cdrDate, err := database.GetMaxEffectiveEndDateForSource(sourceCDR) // MODIFIED HERE
	if err != nil {
		log.Printf("ERROR Service: Failed to get max effective end date for %s from DB: %v\n", sourceCDR, err)
	} else if cdrDate != nil {
		lastKnownEffectiveDates[sourceCDR] = *cdrDate
		log.Printf("INFO Service: Initialized last known %s effective until date from DB: %s\n", sourceCDR, cdrDate.Format("2006-01-02"))
	} else {
		log.Printf("INFO Service: No existing %s effective end date found in DB.\n", sourceCDR)
	}

	// For Preferred Routes
	// Call GetMaxEffectiveEndDateForSource with the generic identifier "PreferredRoutes"
	prefDate, err := database.GetMaxEffectiveEndDateForSource(sourcePreferredRoutes) // MODIFIED HERE
	if err != nil {
		log.Printf("ERROR Service: Failed to get max effective end date for %s from DB: %v\n", sourcePreferredRoutes, err)
	} else if prefDate != nil {
		lastKnownEffectiveDates[sourcePreferredRoutes] = *prefDate
		log.Printf("INFO Service: Initialized last known %s effective until date from DB: %s\n", sourcePreferredRoutes, prefDate.Format("2006-01-02"))
	} else {
		log.Printf("INFO Service: No existing %s effective end date found in DB.\n", sourcePreferredRoutes)
	}
	// TODO: Enhance this to use the `data_source_versions` table for more robust tracking if implemented.
}

// UpdateStaticRouteDataIfNeeded (Code remains the same as the previous version you have)
func UpdateStaticRouteDataIfNeeded(sourceName string, cssSelectorForDate string) error {
	log.Printf("Service: Checking if update is needed for %s data (selector: '%s')...\n", sourceName, cssSelectorForDate)
	log.Println("Service: NOTE - 'UpdateIfNeeded' functionality is currently limited as live 'Effective Date' scraping is deferred pending CSS selector finalization.")

	// Current logic for this function (which defers actual date checking) is fine.
	// It will effectively do nothing until CSS selectors are provided and the commented-out logic is enabled.
	/*
	var currentFAAEffectiveInfo *models.DataSourceEffectiveInfo
	var err error
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

	if !updateNeeded && found { // Optional: Heuristic check
		nextExpectedPublication := lastProcessedUntil.AddDate(0,0, config.AppConfig.DataFreshness.FAAPublicationCycleDays - 7) 
		if time.Now().After(nextExpectedPublication) {
			log.Printf("Service: Approaching/past 56-day cycle for %s. Forcing an update to be safe.", sourceName)
			updateNeeded = true 
		}
	}

	if updateNeeded {
		log.Printf("Service: Update detected as needed for %s based on FAA effective dates.\n", sourceName)
		return ForceUpdateStaticRouteData(sourceName, currentFAAEffectiveInfo) 
	} else {
		log.Printf("Service: No update deemed necessary for %s based on FAA effective dates.\n", sourceName)
	}
	*/
	log.Printf("Service: 'UpdateIfNeeded' for %s concluded (live date check deferred).\n", sourceName)
	return nil
}

// ForceUpdateStaticRouteData (Code remains the same as the previous version you have)
func ForceUpdateStaticRouteData(sourceName string, liveEffectiveInfoFromFAA *models.DataSourceEffectiveInfo) error {
	log.Printf("Service: Forcing update for %s data...\n", sourceName)

	var localPath string
	var csvURL string 
	var downloadFunc func() (string, error)
	var parseFunc func(io.Reader) (interface{}, error)
	var saveFunc func(interface{}, string, *time.Time, *time.Time) error

	var effectiveFrom, effectiveUntil *time.Time
	// var sourceFileForDB string // Declared later

	if liveEffectiveInfoFromFAA != nil {
		effectiveFrom = &liveEffectiveInfoFromFAA.EffectiveFrom
		effectiveUntil = &liveEffectiveInfoFromFAA.EffectiveUntil
		log.Printf("Service: Using provided effective dates for %s: From %s, Until %s\n",
			sourceName, effectiveFrom.Format("2006-01-02"), effectiveUntil.Format("2006-01-02"))
	} else {
		log.Printf("Service: No live effective date info provided for %s. Effective dates in DB will be NULL for this batch unless scraped live (currently deferred).", sourceName)
		// If we strictly don't want to scrape effective dates for manual refresh,
		// then effectiveFrom and effectiveUntil remain nil.
		// The previous version had a section here to scrape them if nil; that part is removed
		// to align with "we don't need to scrape this at all [for manual download]".
	}

	switch sourceName {
	case sourceCDR:
		csvURL = config.AppConfig.FAAURLs.CdrCSV
		downloadFunc = scraper.DownloadCdrCsv
		parseFunc = func(r io.Reader) (interface{}, error) { return scraper.ParseCdrCsv(r) }
		saveFunc = func(data interface{}, sf string, es *time.Time, ee *time.Time) error {
			return database.SaveCdrRoutes(data.([]models.CdrRoute), sf, es, ee)
		}
	case sourcePreferredRoutes:
		csvURL = config.AppConfig.FAAURLs.PreferredRoutesCSV
		downloadFunc = scraper.DownloadPreferredRoutesCsv
		parseFunc = func(r io.Reader) (interface{}, error) { return scraper.ParsePreferredRoutesCsv(r) }
		saveFunc = func(data interface{}, sf string, es *time.Time, ee *time.Time) error {
			return database.SavePreferredRoutes(data.([]models.PreferredRoute), sf, es, ee)
		}
	default:
		return fmt.Errorf("unknown data source name for forced update: %s", sourceName)
	}

	log.Printf("Service: Downloading %s CSV from %s\n", sourceName, csvURL)
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

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded file %s: %w", localPath, err)
	}
	defer file.Close()

	parsedData, err := parseFunc(file)
	if err != nil {
		return fmt.Errorf("failed to parse %s CSV from %s: %w", sourceName, localPath, err)
	}

	sourceFileForDB := filepath.Base(localPath)
	if effectiveFrom != nil {
		sourceFileForDB = fmt.Sprintf("%s_%s", filepath.Base(localPath), effectiveFrom.Format("20060102"))
	}
	
	err = saveFunc(parsedData, sourceFileForDB, effectiveFrom, effectiveUntil)
	if err != nil {
		return fmt.Errorf("failed to save %s routes to database (source file: %s): %w", sourceName, sourceFileForDB, err)
	}

	if effectiveUntil != nil {
		lastKnownEffectiveDates[sourceName] = *effectiveUntil
		log.Printf("Service: Successfully forced update for %s data. New effective until date in memory: %s\n",
			sourceName, effectiveUntil.Format("2006-01-02"))
		// TODO: Persist this update to data_source_versions table
	} else {
		log.Printf("Service: Successfully forced update for %s data. Effective dates are NULL for this batch.\n", sourceName)
		// We might want to clear lastKnownEffectiveDates[sourceName] or handle this state.
		// For now, it means the next "UpdateIfNeeded" will likely see "no previous date" if this was nil.
	}
	return nil
}