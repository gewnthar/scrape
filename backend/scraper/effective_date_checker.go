// backend/scraper/effective_date_checker.go
package scraper

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gewnthar/scrape/backend/config" // For Page URLs
	"github.com/gewnthar/scrape/backend/models"
)

const (
	// This selector should point to a region on the RMT pages that reliably contains the UL with effective dates.
	// 'div.mainArea' is a good candidate based on your inspection.
	// If it's too broad and finds too many ULs, we might need a more specific parent.
	// QC_ACTION: Verify if "div.mainArea" is a suitable container for both RMT pages.
	rmtPageDateContainerSelector = "div.mainArea"

	// Regex to find dates in format "Effective MM/DD/YYYY until MM/DD/YYYY"
	// It captures the 'from' date in group 1 and the 'until' date in group 2.
	effectiveDateRegexString = `Effective\s+(\d{2}/\d{2}/\d{4})\s+until\s+(\d{2}/\d{2}/\d{4})`
	dateLayout               = "01/02/2006" // For parsing MM/DD/YYYY
)

var effectiveDateRegex = regexp.MustCompile(effectiveDateRegexString)

// parseEffectiveDateString extracts 'from' and 'until' dates using the regex.
func parseEffectiveDateString(textToSearch string) (from time.Time, until time.Time, rawMatch string, err error) {
	matches := effectiveDateRegex.FindStringSubmatch(textToSearch)
	if len(matches) < 3 {
		err = fmt.Errorf("could not find full 'Effective ... until ...' pattern in provided text block. Text searched: %s", textToSearch)
		return
	}

	rawMatch = matches[0] // The full matched string "Effective MM/DD/YYYY until MM/DD/YYYY"
	fromString := matches[1]
	untilString := matches[2]

	from, err = time.Parse(dateLayout, fromString)
	if err != nil {
		err = fmt.Errorf("failed to parse 'from' date '%s': %w", fromString, err)
		return
	}

	until, err = time.Parse(dateLayout, untilString)
	if err != nil {
		err = fmt.Errorf("failed to parse 'until' date '%s': %w", untilString, err)
		return
	}
	return
}

// GetEffectiveDatesForDataSource scrapes the given URL, looks for a specific UL structure,
// and extracts effective date information.
func GetEffectiveDatesForDataSource(sourceName, pageURL, containerSelector string) (*models.DataSourceEffectiveInfo, error) {
	log.Printf("Scraper: Checking effective dates for %s from %s (container: '%s')\n", sourceName, pageURL, containerSelector)

	client := http.Client{Timeout: 20 * time.Second}
	res, err := client.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL %s: %w", pageURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get URL %s: status code %d", pageURL, res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML from %s: %w", pageURL, err)
	}

	var foundDateText string
	// Find the container, then iterate through ULs within it
	doc.Find(containerSelector).Find("ul").EachWithBreak(func(i int, ulSelection *goquery.Selection) bool {
		// Check the first list item for "RMT WebService"
		firstLiText := strings.TrimSpace(ulSelection.Find("li:first-of-type").Text())
		if strings.Contains(firstLiText, "RMT WebService") {
			// If found, the second list item should contain the effective dates
			secondLiText := strings.TrimSpace(ulSelection.Find("li:nth-of-type(2)").Text())
			if strings.Contains(secondLiText, "Effective") && strings.Contains(secondLiText, "until") {
				foundDateText = secondLiText
				return false // Stop iterating, we found our target UL
			}
		}
		return true // Continue to the next UL
	})

	if foundDateText == "" {
		log.Printf("WARN Scraper: Could not find the specific UL with 'RMT WebService' and 'Effective ... until ...' structure within container '%s' on page %s.", containerSelector, pageURL)
		// For debugging, you could log the entire container's text:
		// log.Printf("DEBUG: Container HTML for %s on %s: %s", containerSelector, pageURL, doc.Find(containerSelector).Text())
		return nil, fmt.Errorf("target UL for effective dates not found on %s within %s", pageURL, containerSelector)
	}

	from, until, rawStr, err := parseEffectiveDateString(foundDateText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse effective dates for %s from text '%s': %w", sourceName, foundDateText, err)
	}

	log.Printf("Scraper: Found effective dates for %s: From %s, Until %s (Raw: '%s')\n",
		sourceName, from.Format(dateLayout), until.Format(dateLayout), rawStr)

	return &models.DataSourceEffectiveInfo{
		SourceName:     sourceName,
		EffectiveFrom:  from,
		EffectiveUntil: until,
		RawDateString:  rawStr,
		LastChecked:    time.Now().UTC(),
	}, nil
}

// ScrapeEffectiveDatesForCDR fetches effective dates for the CDR data source.
func ScrapeEffectiveDatesForCDR() (*models.DataSourceEffectiveInfo, error) {
	// The specific CSS selector from config is now less critical if the container approach works.
	// We use the configured page URL and a general container selector.
	pageURL := config.AppConfig.FAAURLs.CdrEffectiveDatePage
	// Use the globally defined container or one from config if you make it configurable per source
	container := rmtPageDateContainerSelector 
	// If you stored specific selectors for CDR page in config, you could use:
	// container := config.AppConfig.ScraperSelectors.CdrEffectiveDateContainer (new config field)
	return GetEffectiveDatesForDataSource("CDR", pageURL, container)
}

// ScrapeEffectiveDatesForPreferredRoutes fetches effective dates for the Preferred Routes data source.
func ScrapeEffectiveDatesForPreferredRoutes() (*models.DataSourceEffectiveInfo, error) {
	pageURL := config.AppConfig.FAAURLs.PreferredRoutesEffectiveDatePage
	container := rmtPageDateContainerSelector
	// If you stored specific selectors for Pref Routes page in config, you could use:
	// container := config.AppConfig.ScraperSelectors.PreferredRoutesEffectiveDateContainer (new config field)
	return GetEffectiveDatesForDataSource("PreferredRoutes", pageURL, container)
}