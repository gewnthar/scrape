// backend/models/route.go
package models

import "time"

// CdrRoute represents a Coded Departure Route from codedswap_db.csv
// CSV tags now EXACTLY match the headers you provided.
type CdrRoute struct {
	ID int64 `db:"id"` // For database primary key, not from CSV

	RouteCode            string `csv:"RCode" db:"route_code"`
	Origin               string `csv:"Orig" db:"origin"`
	Destination          string `csv:"Dest" db:"destination"`
	DepartureFix         string `csv:"DepFix" db:"departure_fix"`
	RouteString          string `csv:"Route String" db:"route_string"` // Note: Space in header
	DepartureARTCC       string `csv:"DCNTR" db:"departure_artcc"`
	ArrivalARTCC         string `csv:"ACNTR" db:"arrival_artcc"`
	TraversedARTCCs      string `csv:"TCNTRs" db:"traversed_artccs"`
	CoordinationRequired string `csv:"CoordReq" db:"coordination_required"`
	AssociatedPlay       string `csv:"Play" db:"associated_play"` // CSV header is "Play"
	NavEqp               string `csv:"NavEqp" db:"nav_eqp"`       // CSV header is "NavEqp"

	// Database specific fields
	EffectiveDateStart *time.Time `db:"effective_date_start"`
	EffectiveDateEnd   *time.Time `db:"effective_date_end"`
	SourceFile         string     `db:"source_file"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

// PreferredRoute represents an NFDC Preferred Route from prefroutes_db.csv
// CSV tags now EXACTLY match the headers you provided.
type PreferredRoute struct {
	ID int64 `db:"id"` // For database primary key

	Origin          string `csv:"Orig" db:"origin"`                 // Was "orig"
	RouteString     string `csv:"Route String" db:"route_string"`   // Was "route string"
	Destination     string `csv:"Dest" db:"destination"`            // Was "dest"
	Hours1          string `csv:"Hours1" db:"hours1"`               // Was "hours 1"
	Hours2          string `csv:"Hours2" db:"hours2"`               // Was "hours 2"
	Hours3          string `csv:"Hours3" db:"hours3"`               // Was "hours 3"
	Type            string `csv:"Type" db:"route_type"`             // Was "type"
	Area            string `csv:"Area" db:"area"`                   // Was "area"
	Altitude        string `csv:"Altitude" db:"altitude"`           // Was "alt"
	Aircraft        string `csv:"Aircraft" db:"aircraft"`           // Was "aircraft"
	Direction       string `csv:"Direction" db:"direction"`         // Was "direction"
	Sequence        string `csv:"Seq" db:"sequence"`                // Was "seq"
	DepartureARTCC  string `csv:"DCNTR" db:"departure_artcc"`     // Was "DCNTR" (already correct)
	ArrivalARTCC    string `csv:"ACNTR" db:"arrival_artcc"`         // Was "ACNTR" (already correct)

	// Database specific fields
	EffectiveDateStart *time.Time `db:"effective_date_start"`
	EffectiveDateEnd   *time.Time `db:"effective_date_end"`
	SourceFile         string     `db:"source_file"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

// DataSourceEffectiveInfo holds the scraped effective date range for a data source.
// (This struct remains the same as before)
type DataSourceEffectiveInfo struct {
	SourceName     string    // e.g., "CDR", "PreferredRoutes"
	EffectiveFrom  time.Time
	EffectiveUntil time.Time
	RawDateString  string    // The full "Effective ... until ..." string scraped
	LastChecked    time.Time // When this information was scraped
}

// RecommendedRoute (This struct remains the same as before)
type RecommendedRoute struct {
	Origin          string
	Destination     string
	RouteString     string
	Source          string    
	Justification   string    
	FullAdvisory    *AdvisoryDetail 
	Cdr             *CdrRoute       
	Preferred       *PreferredRoute 
	Priority        int             
	Restrictions    string          
	EffectiveStart  *time.Time      
	EffectiveEnd    *time.Time      
}

// Constants for route priorities (remain the same)
const (
	PriorityRqdAdvisory         = 10
	PriorityRatReroute          = 15 
	PriorityFileCDRsDirective   = 20 
	PriorityCdrNoCoord          = 30
	PriorityPreferredRoute      = 40
	PriorityCdrCoord            = 50
	PriorityInformationalAdvisory = 60
)