// backend/services/advisory_service.go
package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gewnthar/scrape/backend/config" // Adjust to your module path
	"github.com/gewnthar/scrape/backend/database"
	"github.com/gewnthar/scrape/backend/models"
	"github.com/gewnthar/scrape/backend/scraper" // Will call functions from here
)

// --- Scenario A: Exploratory Day Search (00Z to next day 08Z, auto-save all) ---

// ProcessExploratoryAdvisorySearch fetches advisories for a specific operational window
// (day1 00:00Z to day2 08:00Z) and auto-saves both summaries and their full details.
func ProcessExploratoryAdvisorySearch(targetDate time.Time) error {
	log.Printf("Service: Starting exploratory advisory search for target date %s\n", targetDate.Format("2006-01-02"))

	// Define the full 32-hour window based on Zulu time
	// Day 1: targetDate (from 00:00Z)
	// Day 2: targetDate + 1 day (until 08:00Z)
	day1 := targetDate.Truncate(24 * time.Hour)
	day2 := day1.AddDate(0, 0, 1)

	windowStart := day1 // April 1st 00:00Z
	windowEnd := day2.Add(8 * time.Hour) // April 2nd 08:00Z

	log.Printf("Service: Operational window for exploratory search: %s to %s\n",
		windowStart.Format(time.RFC3339), windowEnd.Format(time.RFC3339))

	// Step 1: Fetch summaries for Day 1 and Day 2 to cover the window
	// TODO: Implement scraper.FetchAdvisorySummariesForDate in advisory_scraper.go
	// This function should return []models.AdvisorySummary and handle unique key generation
	// and detail_page_params_json marshalling within the summary objects.
	summariesDay1, err := scraper.FetchAdvisorySummariesForDate(day1, config.AppConfig.FAAURLs.AdvisoryListPageBase)
	if err != nil {
		log.Printf("WARN Service: Failed to fetch summaries for day 1 (%s): %v\n", day1.Format("2006-01-02"), err)
		// Decide if to continue with potentially partial data or return error
	}
	log.Printf("Service: Fetched %d summaries for day 1 (%s)\n", len(summariesDay1), day1.Format("2006-01-02"))

	var summariesDay2 []models.AdvisorySummary
	// Only fetch day2 if windowEnd actually goes into day2
	if windowEnd.After(day2) { // Check if 08:00Z on day2 is part of the window
		summariesDay2, err = scraper.FetchAdvisorySummariesForDate(day2, config.AppConfig.FAAURLs.AdvisoryListPageBase)
		if err != nil {
			log.Printf("WARN Service: Failed to fetch summaries for day 2 (%s): %v\n", day2.Format("2006-01-02"), err)
		}
		log.Printf("Service: Fetched %d summaries for day 2 (%s)\n", len(summariesDay2), day2.Format("2006-01-02"))
	}
	
	allSummaries := append(summariesDay1, summariesDay2...)
	var advisoriesInWindow []models.AdvisorySummary

	// Step 2: Filter summaries to the precise 32-hour window
	for _, summary := range allSummaries {
		// Assuming summary.IssueTimeOnListPage is correctly populated by the scraper and is in UTC
		if summary.IssueTimeOnListPage == nil {
			// If issue time is not on the summary, we might assume it's within the day it was scraped from
			// This logic might need refinement based on actual data from scraper
			summaryDate := summary.AdvisoryDate.Truncate(24 * time.Hour)
			tempIssueTime := time.Date(summaryDate.Year(), summaryDate.Month(), summaryDate.Day(), 0,0,0,0, time.UTC)
			// For simplicity, if no time, check if summary.AdvisoryDate is day1 or day2 within range
			if (summary.AdvisoryDate.Equal(day1) && !tempIssueTime.Before(windowStart)) || 
			   (summary.AdvisoryDate.Equal(day2) && tempIssueTime.Before(windowEnd)) {
				advisoriesInWindow = append(advisoriesInWindow, summary)
			}
			continue
		}
		
		// Construct full issue datetime for comparison
		fullIssueDateTime := time.Date(
			summary.AdvisoryDate.Year(), summary.AdvisoryDate.Month(), summary.AdvisoryDate.Day(),
			summary.IssueTimeOnListPage.Hour(), summary.IssueTimeOnListPage.Minute(), summary.IssueTimeOnListPage.Second(), 0,
			time.UTC, // Assuming issue times are Zulu
		)

		if !fullIssueDateTime.Before(windowStart) && fullIssueDateTime.Before(windowEnd) {
			advisoriesInWindow = append(advisoriesInWindow, summary)
		}
	}
	log.Printf("Service: Found %d advisories within the %s to %s window.\n", len(advisoriesInWindow), windowStart.Format(time.RFC3339), windowEnd.Format(time.RFC3339))


	// Step 3: For each advisory in the window, fetch its detail and save both summary and detail
	if len(advisoriesInWindow) > 0 {
		// First save all summaries (or update them if they already exist)
		// This ensures foreign key constraints are met if details are saved first by unique key
		// and that `has_full_detail_saved` is initially false or its current state.
		// The SaveAdvisoryDetail will then update this flag.
		err = database.SaveAdvisorySummaries(advisoriesInWindow)
		if err != nil {
			log.Printf("ERROR Service: Failed to save some summaries during exploratory search: %v\n", err)
			// Potentially continue to fetch details for successfully saved summaries, or return
		}
	}


	for _, summary := range advisoriesInWindow {
		log.Printf("Service: Processing detail for advisory: %s\n", summary.SummaryUniqueKey)
		// TODO: Implement scraper.FetchAdvisoryDetail in advisory_scraper.go
		// It should take summary.DetailPageParams (map[string]string unmarshalled from JSON)
		// and return *models.AdvisoryDetail, error
		
		// Unmarshal params for scraper
		var detailParams map[string]string
		if err := json.Unmarshal([]byte(summary.DetailPageParamsJSON), &detailParams); err != nil {
			log.Printf("ERROR Service: Could not unmarshal DetailPageParams for %s: %v. Skipping detail fetch.", summary.SummaryUniqueKey, err)
			continue
		}

		detail, err := scraper.FetchAdvisoryDetail(detailParams)
		if err != nil {
			log.Printf("ERROR Service: Failed to fetch detail for summary %s: %v\n", summary.SummaryUniqueKey, err)
			continue // Skip this advisory, try next
		}
		if detail == nil {
			log.Printf("WARN Service: No detail content returned for summary %s.\n", summary.SummaryUniqueKey)
			continue
		}

		// Ensure the detail links back to the summary
		detail.SummaryKey = summary.SummaryUniqueKey
		// Set source, and any other fields if not set by scraper
		if detail.Source == "" {
			detail.Source = "FAA_ADVISORY" // Default source
		}

		// Save the detail (this function also updates summary's has_full_detail_saved flag)
		err = database.SaveAdvisoryDetail(*detail)
		if err != nil {
			log.Printf("ERROR Service: Failed to save detail for summary %s: %v\n", summary.SummaryUniqueKey, err)
			// Continue to next advisory
		} else {
			log.Printf("Service: Successfully fetched and saved detail for summary %s\n", summary.SummaryUniqueKey)
		}
	}

	log.Printf("Service: Exploratory advisory search completed for target date %s.\n", targetDate.Format("2006-01-02"))
	return nil // Return an aggregated error if needed
}


// --- Scenario B: Targeted Search (Day + Keyword, then view/save detail) ---

// GetAndDisplayAdvisorySummaries fetches summaries for a date, potentially scrapes if not in DB,
// and then filters by keyword.
func GetAndDisplayAdvisorySummaries(date time.Time, keyword string) ([]models.AdvisorySummary, error) {
	log.Printf("Service: Getting summaries for date %s, keyword '%s'\n", date.Format("2006-01-02"), keyword)
	
	queryDate := date.Truncate(24 * time.Hour)
	summaries, err := database.GetAdvisorySummariesByDate(queryDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get summaries from DB: %w", err)
	}

	// Simple "stale" check: if it's today and no summaries, or if you want to refresh today's data periodically.
	// For past dates, we generally trust what's in the DB unless a force refresh is triggered.
	isToday := queryDate.Equal(time.Now().UTC().Truncate(24 * time.Hour))
	if len(summaries) == 0 || isToday { // Always scrape for today, or if no data for a past day
		log.Printf("Service: No summaries in DB for %s or it's today. Fetching from FAA...\n", queryDate.Format("2006-01-02"))
		// TODO: Implement scraper.FetchAdvisorySummariesForDate
		fetchedSummaries, scrapeErr := scraper.FetchAdvisorySummariesForDate(queryDate, config.AppConfig.FAAURLs.AdvisoryListPageBase)
		if scrapeErr != nil {
			if len(summaries) > 0 { // If scrape failed but we have old data, log and return old
				log.Printf("WARN Service: Failed to fetch live summaries for %s (%v), returning stale data from DB.\n", queryDate.Format("2006-01-02"), scrapeErr)
			} else { // No old data and scrape failed
				return nil, fmt.Errorf("failed to fetch live summaries for %s and no stale data available: %w", queryDate.Format("2006-01-02"), scrapeErr)
			}
		} else {
			if len(fetchedSummaries) > 0 {
				// Save newly fetched summaries
				// The SaveAdvisorySummaries uses ON DUPLICATE KEY UPDATE, so it's safe.
				// It will also update last_seen_at for existing summaries.
				dbErr := database.SaveAdvisorySummaries(fetchedSummaries)
				if dbErr != nil {
					log.Printf("ERROR Service: Failed to save newly fetched summaries: %v\n", dbErr)
					// Fallback to using potentially unsaved fetchedSummaries for current request if DB save fails
				}
				summaries = fetchedSummaries // Use the freshly scraped data
				log.Printf("Service: Successfully fetched and saved/updated %d live summaries for %s.\n", len(summaries), queryDate.Format("2006-01-02"))
			} else if len(summaries) == 0 { // Scrape returned no summaries, and DB had none
                log.Printf("Service: No advisories found on FAA site for %s.\n", queryDate.Format("2006-01-02"))
            }
		}
	}

	// Filter by keyword if provided
	if keyword != "" {
		var filteredSummaries []models.AdvisorySummary
		lowerKeyword := strings.ToLower(keyword)
		for _, summary := range summaries {
			if strings.Contains(strings.ToLower(summary.ListPageRawText), lowerKeyword) {
				filteredSummaries = append(filteredSummaries, summary)
			}
		}
		log.Printf("Service: Filtered %d summaries down to %d by keyword '%s'\n", len(summaries), len(filteredSummaries), keyword)
		return filteredSummaries, nil
	}

	return summaries, nil
}

// GetOrFetchAdvisoryDetail tries to get a detail from DB; if not found, scrapes it from FAA.
// This function returns the detail *without saving it*. Saving is a separate step.
// `summaryKey` is used to check DB. `detailParams` (from summary.DetailPageParams) is used to scrape if not in DB.
func GetOrFetchAdvisoryDetail(summaryKey string, detailParams map[string]string) (*models.AdvisoryDetail, error) {
	log.Printf("Service: Getting or fetching detail for summary_key: %s\n", summaryKey)

	// Try fetching from DB first
	detail, err := database.GetAdvisoryDetail(summaryKey)
	if err != nil {
		return nil, fmt.Errorf("error checking database for advisory detail %s: %w", summaryKey, err)
	}
	if detail != nil {
		log.Printf("Service: Found advisory detail for %s in database.\n", summaryKey)
		return detail, nil
	}

	// Not in DB, so scrape it from FAA
	log.Printf("Service: Advisory detail for %s not in DB. Scraping from FAA...\n", summaryKey)
	if detailParams == nil {
		return nil, fmt.Errorf("cannot scrape detail for %s: detailParams are nil", summaryKey)
	}

	// TODO: Implement scraper.FetchAdvisoryDetail
	scrapedDetail, err := scraper.FetchAdvisoryDetail(detailParams)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape advisory detail for %s: %w", summaryKey, err)
	}
	if scrapedDetail == nil {
		log.Printf("WARN Service: Scraper returned no detail content for summary %s.\n", summaryKey)
		return nil, nil // Or an error indicating not found on FAA site
	}

	// Ensure the detail links back to the summary and has a source
	scrapedDetail.SummaryKey = summaryKey
	if scrapedDetail.Source == "" {
		scrapedDetail.Source = "FAA_ADVISORY" // Default source
	}
	
	log.Printf("Service: Successfully scraped (but not saved) advisory detail for %s.\n", summaryKey)
	return scrapedDetail, nil
}

// ConfirmSaveAdvisoryDetail saves a provided AdvisoryDetail to the database.
// This is called after the user has viewed a scraped detail and confirms they want to save it.
func ConfirmSaveAdvisoryDetail(detail models.AdvisoryDetail) error {
	if detail.SummaryKey == "" {
		return fmt.Errorf("cannot save advisory detail: SummaryKey is empty")
	}
	log.Printf("Service: Confirming save for advisory detail: %s\n", detail.SummaryKey)
	err := database.SaveAdvisoryDetail(detail)
	if err != nil {
		return fmt.Errorf("failed to save confirmed advisory detail for %s: %w", detail.SummaryKey, err)
	}
	log.Printf("Service: Successfully saved advisory detail for %s to database.\n", detail.SummaryKey)
	return nil
}