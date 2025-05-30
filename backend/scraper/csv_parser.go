// backend/scraper/csv_parser.go
package scraper

import (
	"fmt"
	"io"
	"log"
	"os" // Only for the example testing function, not for main parsing logic

	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
	"github.com/jszwec/csvutil"
)

// ParseCdrCsv takes an io.Reader containing CSV data for CDRs
// and returns a slice of CdrRoute structs.
func ParseCdrCsv(reader io.Reader) ([]models.CdrRoute, error) {
	var cdrRoutes []models.CdrRoute

	// Create a new CSV decoder.
	// csvutil assumes the first line is a header and uses it to map to struct fields
	// based on the `csv:"..."` tags in models.CdrRoute.
	decoder, err := csvutil.NewDecoder(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV decoder for CDRs: %w", err)
	}

	// Configure decoder if needed (e.g., if delimiter is not a comma, or other settings)
	// For now, defaults are usually fine. Ensure your CSV headers EXACTLY match your struct tags.

	// Read all records
	if err := decoder.Decode(&cdrRoutes); err != nil {
		return nil, fmt.Errorf("failed to decode CDR CSV data: %w", err)
	}

	log.Printf("Successfully parsed %d CDR routes from CSV.\n", len(cdrRoutes))
	return cdrRoutes, nil
}

// ParsePreferredRoutesCsv takes an io.Reader containing CSV data for Preferred Routes
// and returns a slice of PreferredRoute structs.
func ParsePreferredRoutesCsv(reader io.Reader) ([]models.PreferredRoute, error) {
	var preferredRoutes []models.PreferredRoute

	decoder, err := csvutil.NewDecoder(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV decoder for Preferred Routes: %w", err)
	}

	// Ensure your CSV headers for prefroutes_db.csv EXACTLY match your struct tags in models.PreferredRoute.

	if err := decoder.Decode(&preferredRoutes); err != nil {
		return nil, fmt.Errorf("failed to decode Preferred Routes CSV data: %w", err)
	}

	log.Printf("Successfully parsed %d Preferred Routes from CSV.\n", len(preferredRoutes))
	return preferredRoutes, nil
}

// --- Example Usage / Testing Function (Optional - for local testing) ---
// You can uncomment this and call it from a temporary main or a test file
// to verify parsing with your local CSV files.
/*
func TestParseLocalCSVs() {
	// Test CDRs
	cdrFile, err := os.Open("../../codedswap_db.csv") // Adjust path to where your CSV is
	if err != nil {
		log.Fatalf("Failed to open codedswap_db.csv for testing: %v", err)
	}
	defer cdrFile.Close()

	cdrRoutes, err := ParseCdrCsv(cdrFile)
	if err != nil {
		log.Fatalf("Error parsing CDR CSV for testing: %v", err)
	}
	if len(cdrRoutes) > 0 {
		log.Printf("TestParse: First CDR Route: %+v\n", cdrRoutes[0])
		log.Printf("TestParse: Parsed %d CDR routes successfully.\n", len(cdrRoutes))
	} else {
		log.Println("TestParse: No CDR routes parsed or CSV empty.")
	}

	// Test Preferred Routes
	prefFile, err := os.Open("../../prefroutes_db.csv") // Adjust path to where your CSV is
	if err != nil {
		log.Fatalf("Failed to open prefroutes_db.csv for testing: %v", err)
	}
	defer prefFile.Close()

	prefRoutes, err := ParsePreferredRoutesCsv(prefFile)
	if err != nil {
		log.Fatalf("Error parsing Preferred Routes CSV for testing: %v", err)
	}
	if len(prefRoutes) > 0 {
		log.Printf("TestParse: First Preferred Route: %+v\n", prefRoutes[0])
		log.Printf("TestParse: Parsed %d Preferred Routes successfully.\n", len(prefRoutes))

	} else {
		log.Println("TestParse: No Preferred Routes parsed or CSV empty.")
	}
}
*/