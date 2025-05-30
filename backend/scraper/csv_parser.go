// backend/scraper/csv_parser.go
package scraper

import (
	"fmt"
	"io"
	"log"
	// "os" // Only for TestParseLocalCSVs if you use it

	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
	"github.com/jszwec/csvutil"
)

// ParseCdrCsv takes an io.Reader containing CSV data for CDRs
// and returns a slice of CdrRoute structs.
func ParseCdrCsv(reader io.Reader) ([]models.CdrRoute, error) {
	var cdrRoutes []models.CdrRoute

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data for CDRs: %w", err)
	}

	// csvutil.Unmarshal uses the `csv:"..."` tags in models.CdrRoute
	// which should now EXACTLY match your CSV headers.
	if err := csvutil.Unmarshal(data, &cdrRoutes); err != nil {
		previewLength := 300 // Show a bit more for context
		if len(data) < previewLength {
			previewLength = len(data)
		}
		log.Printf("ERROR unmarshalling CDR CSV. Data preview (first %d bytes):\n%s\n", previewLength, string(data[:previewLength]))
		return nil, fmt.Errorf("failed to unmarshal CDR CSV data: %w", err)
	}

	log.Printf("Successfully parsed %d CDR routes from CSV.\n", len(cdrRoutes))
	// Log first few parsed routes for QC to see if fields are populated
	for i := 0; i < 5 && i < len(cdrRoutes); i++ {
		log.Printf("DIAGNOSTIC Parsed CDR [%d]: Origin: '%s', Dest: '%s', Code: '%s', Route: '%s'\n",
			i, cdrRoutes[i].Origin, cdrRoutes[i].Destination, cdrRoutes[i].RouteCode, cdrRoutes[i].RouteString)
	}
	if len(cdrRoutes) == 0 {
		log.Println("WARN: Parsed 0 CDR routes. Check CSV content and model tags.")
	}
	return cdrRoutes, nil
}

// ParsePreferredRoutesCsv takes an io.Reader containing CSV data for Preferred Routes
// and returns a slice of PreferredRoute structs.
func ParsePreferredRoutesCsv(reader io.Reader) ([]models.PreferredRoute, error) {
	var preferredRoutes []models.PreferredRoute

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data for Preferred Routes: %w", err)
	}

	// csvutil.Unmarshal uses the `csv:"..."` tags in models.PreferredRoute
	// which should now EXACTLY match your CSV headers.
	if err := csvutil.Unmarshal(data, &preferredRoutes); err != nil {
		previewLength := 300 // Show a bit more for context
		if len(data) < previewLength {
			previewLength = len(data)
		}
		log.Printf("ERROR unmarshalling Preferred Routes CSV. Data preview (first %d bytes):\n%s\n", previewLength, string(data[:previewLength]))
		return nil, fmt.Errorf("failed to unmarshal Preferred Routes CSV data: %w", err)
	}

	log.Printf("Successfully parsed %d Preferred Routes from CSV.\n", len(preferredRoutes))
	// Log first few parsed routes for QC to see if fields are populated
	for i := 0; i < 5 && i < len(preferredRoutes); i++ {
		log.Printf("DIAGNOSTIC Parsed Preferred Route [%d]: Origin: '%s', Dest: '%s', Type: '%s', Route: '%s'\n",
			i, preferredRoutes[i].Origin, preferredRoutes[i].Destination, preferredRoutes[i].Type, preferredRoutes[i].RouteString)
	}
	if len(preferredRoutes) == 0 {
		log.Println("WARN: Parsed 0 Preferred routes. Check CSV content and model tags.")
	}
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