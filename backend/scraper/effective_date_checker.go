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
	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
)

const (
	// QC: YOU MUST VERIFY AND UPDATE THESE URLs AND SELECTORS!
	cdrQueryPageURL          = "https://www.fly.faa.gov/rmt/cdm_operational_coded_departur.jsp"
	preferredRoutesQueryPageURL = "https://www.fly.faa.gov/rmt/nfdc_preferred_routes_database.jsp"

	// QC: Example selector - find the actual CSS selector for the element containing the date string.
	// It might be a <p>, <span>, <div> with a specific class or ID, or within a table.
	// For example, if the text is in <p class="effective-dates">Effective ...</p>
	// the selector would be "p.effective-dates"
	// THIS IS A GUESS - PLEASE UPDATE:
	defaultDateStringSelector = "body" // Fallback, search whole body - VERY INEFFICIENT, PLEASE REFINE

	// Regex to find dates in format "Effective MM/DD/YYYY until MM/DD/YYYY"
	// It captures the 'from' date in group 1 and the 'until' date in group 2.
	effectiveDateRegexString = `Effective\s+(\d{2}/\d{2}/\d{4})\s+until\s+(\d{2}/\d{2}/\d{4})`
	dateLayout               = "01/02/2006" // For parsing MM/DD/YYYY
)

var effectiveDateRegex = regexp.MustCompile(effectiveDateRegexString)

// fetchPageAndFindText fetches a URL and uses a goquery selector to find text.
func fetchPageAndFindText(url string, selector string) (string, error) {
	client := http.Client{Timeout: 20 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get URL %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get URL %s: status code %d", url, res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML from %s: %w", url, err)
	}

	// Find the text using the selector.
	// If selector is broad (like "body"), it might return a lot of text.
	// A more specific selector is highly recommended.
	foundText := doc.Find(selector).First().Text()
	if foundText == "" {
		// Try searching the whole body if specific selector yields nothing
		// but warn if the specific selector was not the default "body"
		if selector != "body" {
			log.Printf("Warning: CSS selector '%s' yielded no text on %s. Falling back to searching body.", selector, url)
			foundText = doc.Find("body").Text()
		}
		if foundText == "" {
			return "", fmt.Errorf("no text found with selector '%s' (or in body) on page %s", selector, url)
		}
	}
	return strings.TrimSpace(foundText), nil
}

// parseEffectiveDateString extracts 'from' and 'until' dates using the regex.
func parseEffectiveDateString(text string) (from time.Time, until time.Time, rawMatch string, err error) {
	matches := effectiveDateRegex.FindStringSubmatch(text)
	if len(matches) < 3 {
		// If the main regex doesn't match, try to find just one date if possible,
		// or look for simpler patterns if the FAA format varies.
		// For now, we require the full "Effective ... until ..." pattern.
		err = fmt.Errorf("could not find full 'Effective ... until ...' pattern in text: %s", text)
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

// GetEffectiveDatesForDataSource scrapes the given URL and selector for effective date information.
func GetEffectiveDatesForDataSource(sourceName, url, cssSelector string) (*models.DataSourceEffectiveInfo, error) {
	log.Printf("Checking effective dates for %s from %s\n", sourceName, url)
	
	// Use defaultDateStringSelector if a specific one isn't provided or is empty
	actualSelector := cssSelector
	if actualSelector == "" {
		actualSelector = defaultDateStringSelector
	}

	pageText, err := fetchPageAndFindText(url, actualSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch or find text for %s: %w", sourceName, err)
	}

	from, until, rawStr, err := parseEffectiveDateString(pageText)
	if err != nil {
		// Log the text we tried to parse for easier debugging by the user
		log.Printf("Full text searched for date pattern in %s: %s\n", sourceName, pageText)
		return nil, fmt.Errorf("failed to parse effective dates for %s: %w", sourceName, err)
	}

	log.Printf("Found effective dates for %s: From %s, Until %s (Raw: '%s')\n",
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
// QC: You may need to provide a more specific CSS selector for cdrDateStringSelector.
func ScrapeEffectiveDatesForCDR(cdrDateStringSelector string) (*models.DataSourceEffectiveInfo, error) {
	return GetEffectiveDatesForDataSource("CDR", cdrQueryPageURL, cdrDateStringSelector)
}

// ScrapeEffectiveDatesForPreferredRoutes fetches effective dates for the Preferred Routes data source.
// QC: You may need to provide a more specific CSS selector for prefRoutesDateStringSelector.
func ScrapeEffectiveDatesForPreferredRoutes(prefRoutesDateStringSelector string) (*models.DataSourceEffectiveInfo, error) {
	return GetEffectiveDatesForDataSource("PreferredRoutes", preferredRoutesQueryPageURL, prefRoutesDateStringSelector)
}