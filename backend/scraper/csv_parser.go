// backend/scraper/csv_parser.go
package scraper

import (
	"fmt"
	"io" // Make sure io is imported for io.ReadAll
	"log"

	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
	"github.com/jszwec/csvutil"
)

// ParseCdrCsv takes an io.Reader containing CSV data for CDRs
// and returns a slice of CdrRoute structs.
func ParseCdrCsv(reader io.Reader) ([]models.CdrRoute, error) {
	var cdrRoutes []models.CdrRoute

	// Read all data from the reader first
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data for CDRs: %w", err)
	}

	// Unmarshal the data
	if err := csvutil.Unmarshal(data, &cdrRoutes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CDR CSV data: %w", err)
	}

	log.Printf("Successfully parsed %d CDR routes from CSV.\n", len(cdrRoutes))
	return cdrRoutes, nil
}

// ParsePreferredRoutesCsv takes an io.Reader containing CSV data for Preferred Routes
// and returns a slice of PreferredRoute structs.
func ParsePreferredRoutesCsv(reader io.Reader) ([]models.PreferredRoute, error) {
	var preferredRoutes []models.PreferredRoute

	// Read all data from the reader first
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed_to_read_csv_data_for_preferred_routes: %w", err)
	}

	// Unmarshal the data
	if err := csvutil.Unmarshal(data, &preferredRoutes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Preferred Routes CSV data: %w", err)
	}

	log.Printf("Successfully parsed %d Preferred Routes from CSV.\n", len(preferredRoutes))
	return preferredRoutes, nil
}

// --- Example Usage / Testing Function (Optional - for local testing) ---
// (TestParseLocalCSVs function remains the same as before)
/*
func TestParseLocalCSVs() {
	// Test CDRs
	cdrFile, err := os.Open("../../codedswap_db.csv") 
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
	prefFile, err := os.Open("../../prefroutes_db.csv") 
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