// backend/scraper/csv_parser.go
package scraper

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
	"github.com/jszwec/csvutil"
)

// normalizeAirportCode converts 4-letter US ICAO codes (e.g., "KJFK") to 3-letter codes ("JFK").
// Other codes are returned as is. Converts to uppercase.
func normalizeAirportCode(code string) string {
	upperCode := strings.ToUpper(strings.TrimSpace(code))
	if len(upperCode) == 4 && strings.HasPrefix(upperCode, "K") {
		// Basic check for US K-prefixed ICAO codes
		// More sophisticated logic might be needed for global codes or if non-K 4-letter codes exist.
		return upperCode[1:]
	}
	return upperCode
}

// ParseCdrCsv takes an io.Reader containing CSV data for CDRs
// and returns a slice of CdrRoute structs, with airport codes normalized.
func ParseCdrCsv(reader io.Reader) ([]models.CdrRoute, error) {
	var cdrRoutes []models.CdrRoute

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data for CDRs: %w", err)
	}

	// IMPORTANT: Ensure csv:"..." tags in models.CdrRoute EXACTLY match CSV headers.
	// Current tags are based on: RCode,Orig,Dest,DepFix,Route String,DCNTR,ACNTR,TCNTRs,CoordReq,Play,NavEqp
	if err := csvutil.Unmarshal(data, &cdrRoutes); err != nil {
		previewLength := 300
		if len(data) < previewLength {
			previewLength = len(data)
		}
		log.Printf("ERROR unmarshalling CDR CSV. Data preview (first %d bytes):\n%s\n", previewLength, string(data[:previewLength]))
		return nil, fmt.Errorf("failed to unmarshal CDR CSV data: %w", err)
	}

	// Normalize airport codes after parsing
	for i := range cdrRoutes {
		cdrRoutes[i].Origin = normalizeAirportCode(cdrRoutes[i].Origin)
		cdrRoutes[i].Destination = normalizeAirportCode(cdrRoutes[i].Destination)
		// Potentially normalize other airport code fields if they exist and need it
	}

	log.Printf("Successfully parsed and normalized %d CDR routes from CSV.\n", len(cdrRoutes))
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
// and returns a slice of PreferredRoute structs, with airport codes normalized.
func ParsePreferredRoutesCsv(reader io.Reader) ([]models.PreferredRoute, error) {
	var preferredRoutes []models.PreferredRoute

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data for Preferred Routes: %w", err)
	}

	// IMPORTANT: Ensure csv:"..." tags in models.PreferredRoute EXACTLY match CSV headers.
	// Current tags are based on: Orig,Route String,Dest,Hours1,Hours2,Hours3,Type,Area,Altitude,Aircraft,Direction,Seq,DCNTR,ACNTR
	if err := csvutil.Unmarshal(data, &preferredRoutes); err != nil {
		previewLength := 300
		if len(data) < previewLength {
			previewLength = len(data)
		}
		log.Printf("ERROR unmarshalling Preferred Routes CSV. Data preview (first %d bytes):\n%s\n", previewLength, string(data[:previewLength]))
		return nil, fmt.Errorf("failed to unmarshal Preferred Routes CSV data: %w", err)
	}
	
	// Normalize airport codes after parsing
	for i := range preferredRoutes {
		preferredRoutes[i].Origin = normalizeAirportCode(preferredRoutes[i].Origin)
		preferredRoutes[i].Destination = normalizeAirportCode(preferredRoutes[i].Destination)
	}


	log.Printf("Successfully parsed and normalized %d Preferred Routes from CSV.\n", len(preferredRoutes))
	for i := 0; i < 5 && i < len(preferredRoutes); i++ {
		log.Printf("DIAGNOSTIC Parsed Preferred Route [%d]: Origin: '%s', Dest: '%s', Type: '%s', Route: '%s'\n",
			i, preferredRoutes[i].Origin, preferredRoutes[i].Destination, preferredRoutes[i].Type, preferredRoutes[i].RouteString)
	}
	if len(preferredRoutes) == 0 {
		log.Println("WARN: Parsed 0 Preferred routes. Check CSV content and model tags.")
	}
	return preferredRoutes, nil
}

// (TestParseLocalCSVs function can remain as before for your local testing,
// just ensure the paths in it point to your `actual_...csv` files if you use it,
// and that you've updated models/route.go with the correct CSV tags from last step)
/*
func TestParseLocalCSVs() {
	// Test CDRs
	// You'd need to have actual_codedswap_db.csv in the project root, or adjust path
	cdrFile, err := os.Open("../../actual_codedswap_db.csv") 
	if err != nil {
		log.Fatalf("Failed to open actual_codedswap_db.csv for testing: %v", err)
	}
	defer cdrFile.Close()

	cdrRoutes, err := ParseCdrCsv(cdrFile)
	if err != nil {
		log.Fatalf("Error parsing CDR CSV for testing: %v", err)
	}
	if len(cdrRoutes) > 0 {
		// Log a few more details to check normalization
		for i := 0; i < 5 && i < len(cdrRoutes); i++ {
			log.Printf("TestParse CDR [%d]: Orig: '%s', Dest: '%s', Code: '%s'\n", i, cdrRoutes[i].Origin, cdrRoutes[i].Destination, cdrRoutes[i].RouteCode)
		}
	} else {
		log.Println("TestParse: No CDR routes parsed or CSV empty.")
	}

	// Test Preferred Routes
	prefFile, err := os.Open("../../actual_prefroutes_db.csv") 
	if err != nil {
		log.Fatalf("Failed to open actual_prefroutes_db.csv for testing: %v", err)
	}
	defer prefFile.Close()

	prefRoutes, err := ParsePreferredRoutesCsv(prefFile)
	if err != nil {
		log.Fatalf("Error parsing Preferred Routes CSV for testing: %v", err)
	}
	if len(prefRoutes) > 0 {
		for i := 0; i < 5 && i < len(prefRoutes); i++ {
			log.Printf("TestParse PrefRoute [%d]: Orig: '%s', Dest: '%s', Type: '%s'\n", i, prefRoutes[i].Origin, prefRoutes[i].Destination, prefRoutes[i].Type)
		}
	} else {
		log.Println("TestParse: No Preferred Routes parsed or CSV empty.")
	}
}
*/