// backend/models/route.go
package models

import "time"

// CdrRoute represents a Coded Departure Route from the codedswap_db.csv
// The `csv:"..."` tags MUST exactly match your CSV headers for correct parsing.
// Please verify these against the first line of your codedswap_db.csv file.
type CdrRoute struct {
	ID int64 `db:"id"` // For database primary key, not from CSV

	RouteCode            string `csv:"ROUTE_CODE" db:"route_code"`                       // e.g., "JFKMIA48K"
	Origin               string `csv:"ORIGIN" db:"origin"`                               // e.g., "JFK"
	Destination          string `csv:"DESTINATION" db:"destination"`                     // e.g., "MIA"
	DepartureFix         string `csv:"DEPARTURE_FIX" db:"departure_fix"`                 // e.g., "RBV"
	RouteString          string `csv:"ROUTE_STRING" db:"route_string"`                   // e.g., "JFK RBV Q430..."
	DepartureARTCC       string `csv:"DEPARTURE_ARTCC" db:"departure_artcc"`             // e.g., "ZNY"
	ArrivalARTCC         string `csv:"ARRIVAL_ARTCC" db:"arrival_artcc"`                 // e.g., "ZMA"
	TraversedARTCCs      string `csv:"TRAVERSED_ARTCCS" db:"traversed_artccs"`           // e.g., "ZDC ZJX ZMA ZNY ZTL"
	CoordinationRequired string `csv:"COORDINATION_REQUIRED" db:"coordination_required"` // Expect "Y" or "N" from CSV
	AssociatedPlay       string `csv:"ASSOCIATED_PLAY" db:"associated_play"`             // MOVED: This field comes before NavEqp in your CSV
	NavEqp               string `csv:"NAV_EQP" db:"nav_eqp"`                             // MOVED: This field is last among these in your CSV

	// Database specific fields (not from CSV, but good to have)
	EffectiveDateStart *time.Time `db:"effective_date_start"` // If available from filename or metadata
	EffectiveDateEnd   *time.Time `db:"effective_date_end"`   // If available
	SourceFile         string     `db:"source_file"`          // e.g., "codedswap_db.csv_YYYY-MM-DD"
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

// PreferredRoute represents an NFDC Preferred Route from prefroutes_db.csv
// The `csv:"..."` tags MUST exactly match your CSV headers for correct parsing.
type PreferredRoute struct {
	ID int64 `db:"id"` // For database primary key

	Origin          string `csv:"orig" db:"origin"`
	RouteString     string `csv:"route string" db:"route_string"` // Note: CSV header has a space
	Destination     string `csv:"dest" db:"destination"`
	Hours1          string `csv:"hours 1" db:"hours1"`       // Note: CSV header has a space
	Hours2          string `csv:"hours 2" db:"hours2"`       // Note: CSV header has a space
	Hours3          string `csv:"hours 3" db:"hours3"`       // Note: CSV header has a space
	Type            string `csv:"type" db:"route_type"`      // Renamed db field slightly for clarity
	Area            string `csv:"area" db:"area"`
	Altitude        string `csv:"alt" db:"altitude"`         // Renamed db field slightly for clarity ('alt' is a keyword in some SQL contexts)
	Aircraft        string `csv:"aircraft" db:"aircraft"`
	Direction       string `csv:"direction" db:"direction"`
	Sequence        string `csv:"seq" db:"sequence"`         // Renamed db field slightly for clarity
	DepartureARTCC  string `csv:"DCNTR" db:"departure_artcc"`
	ArrivalARTCC    string `csv:"ACNTR" db:"arrival_artcc"`

	// Database specific fields
	// These dates are for the entire file, will be populated during ingestion
	EffectiveDateStart *time.Time `db:"effective_date_start"`
	EffectiveDateEnd   *time.Time `db:"effective_date_end"`
	SourceFile         string     `db:"source_file"` // e.g., "prefroutes_db.csv_YYYY-MM-DD"
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

// DataSourceEffectiveInfo holds the scraped effective date range for a data source.
type DataSourceEffectiveInfo struct {
	SourceName     string    // e.g., "CDR", "PreferredRoutes"
	EffectiveFrom  time.Time
	EffectiveUntil time.Time
	RawDateString  string    // The full "Effective ... until ..." string scraped
	LastChecked    time.Time // When this information was scraped
}


// RecommendedRoute holds a processed route with its source and justification for display.
type RecommendedRoute struct {
	Origin          string
	Destination     string
	RouteString     string
	Source          string    // e.g., "RQD Advisory", "CDR (No Coord)", "Preferred Route", "RAT Reroute"
	Justification   string    // Reason for this route being chosen/listed (e.g., Advisory text, CDR code)
	FullAdvisory    *AdvisoryDetail // If derived from an advisory, link to it
	Cdr             *CdrRoute       // If it's a CDR
	Preferred       *PreferredRoute // If it's a Preferred Route
	Priority        int             // Lower is higher priority
	Restrictions    string          // Any relevant restrictions or remarks
	EffectiveStart  *time.Time      // For display
	EffectiveEnd    *time.Time      // For display
}

// Constants for route priorities (lower is better)
const (
	PriorityRqdAdvisory         = 10
	PriorityRatReroute          = 15 // Specific reroute from RAT reader
	PriorityFileCDRsDirective   = 20 // When an advisory says "File CDRs"
	PriorityCdrNoCoord          = 30
	PriorityPreferredRoute      = 40
	PriorityCdrCoord            = 50
	PriorityInformationalAdvisory = 60
)