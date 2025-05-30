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

// const (
//	// rmtPageDateContainerSelector = "div.mainArea" // This can be removed if selectors are always passed in
// )

// Regex to find dates in format "Effective MM/DD/YYYY until MM/DD/YYYY"
var effectiveDateRegex = regexp.MustCompile(`Effective\s+(\d{2}/\d{2}/\d{4})\s+until\s+(\d{2}/\d{2}/\d{4})`)
const dateLayout  = "01/02/2006" // For parsing MM/DD/YYYY


// parseEffectiveDateString extracts 'from' and 'until' dates using the regex.
// (This function remains the same as before)
func parseEffectiveDateString(textToSearch string) (from time.Time, until time.Time, rawMatch string, err error) {
	matches := effectiveDateRegex.FindStringSubmatch(textToSearch)
	if len(matches) < 3 {
		err = fmt.Errorf("could not find full 'Effective ... until ...' pattern in provided text block. Text searched: %s", textToSearch)
		return
	}

	rawMatch = matches[0] 
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

// GetEffectiveDatesForDataSource scrapes the given URL, looks for a specific UL structure within the container,
// and extracts effective date information.
// (This function remains the same as before)
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
	doc.Find(containerSelector).Find("ul").EachWithBreak(func(i int, ulSelection *goquery.Selection) bool {
		firstLiText := strings.TrimSpace(ulSelection.Find("li:first-of-type").Text())
		if strings.Contains(firstLiText, "RMT WebService") {
			secondLiText := strings.TrimSpace(ulSelection.Find("li:nth-of-type(2)").Text())
			if strings.Contains(secondLiText, "Effective") && strings.Contains(secondLiText, "until") {
				foundDateText = secondLiText
				return false 
			}
		}
		return true 
	})

	if foundDateText == "" {
		log.Printf("WARN Scraper: Could not find the specific UL with 'RMT WebService' and 'Effective ... until ...' structure within container '%s' on page %s.", containerSelector, pageURL)
		// For debugging, log the container's text to see what was searched
		// containerHTML, _ := doc.Find(containerSelector).Html()
		// log.Printf("DEBUG: Container HTML for %s on %s: %s", containerSelector, pageURL, containerHTML)
		return nil, fmt.Errorf("target UL for effective dates not found on %s within container '%s'. QC: Verify container selector and page structure.", pageURL, containerSelector)
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
// It now accepts the containerSelector (typically from config) to use.
func ScrapeEffectiveDatesForCDR(containerSelector string) (*models.DataSourceEffectiveInfo, error) { // MODIFIED: Added parameter
	pageURL := config.AppConfig.FAAURLs.CdrEffectiveDatePage
	if containerSelector == "" { // Fallback if an empty selector is passed
		log.Println("WARN Scraper: No specific CSS selector provided for CDR effective date container, using default 'body'. QC: This is likely inefficient/incorrect.")
		containerSelector = "body" 
	}
	return GetEffectiveDatesForDataSource("CDR", pageURL, containerSelector)
}

// ScrapeEffectiveDatesForPreferredRoutes fetches effective dates for the Preferred Routes data source.
// It now accepts the containerSelector (typically from config) to use.
func ScrapeEffectiveDatesForPreferredRoutes(containerSelector string) (*models.DataSourceEffectiveInfo, error) { // MODIFIED: Added parameter
	pageURL := config.AppConfig.FAAURLs.PreferredRoutesEffectiveDatePage
	if containerSelector == "" { // Fallback if an empty selector is passed
		log.Println("WARN Scraper: No specific CSS selector provided for Preferred Routes effective date container, using default 'body'. QC: This is likely inefficient/incorrect.")
		containerSelector = "body"
	}
	return GetEffectiveDatesForDataSource("PreferredRoutes", pageURL, containerSelector)
}