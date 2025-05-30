// backend/database/static_route_store.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
)

// SaveCdrRoutes saves a slice of CdrRoute objects to the database.
// MODIFIED: Uses a "clear and load" strategy for a given sourceFile.
func SaveCdrRoutes(routes []models.CdrRoute, sourceFile string, effectiveStart, effectiveEnd *time.Time) error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	if len(routes) == 0 {
		log.Println("No CDR routes provided to save.")
		return nil
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for CDR routes: %w", err)
	}
	defer tx.Rollback() 

	// Step 1: Delete existing routes for this sourceFile.
	_, err = tx.Exec("DELETE FROM cdm_routes WHERE source_file = ?", sourceFile)
	if err != nil {
		return fmt.Errorf("failed to delete old CDR routes for source %s: %w", sourceFile, err)
	}
	log.Printf("Cleared existing CDR routes for source: %s\n", sourceFile)

	// Step 2: Insert new routes
	stmt, err := tx.Prepare(`
		INSERT INTO cdm_routes (
			route_code, origin, destination, departure_fix, route_string,
			departure_artcc, arrival_artcc, traversed_artccs,
			coordination_required, nav_eqp, associated_play,
			effective_date_start, effective_date_end, source_file, updated_at 
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
	`) 
	if err != nil {
		return fmt.Errorf("failed to prepare CDR insert statement: %w", err)
	}
	defer stmt.Close()

	for _, route := range routes {
		var sqlEffectiveStart, sqlEffectiveEnd sql.NullTime
		if effectiveStart != nil {
			sqlEffectiveStart = sql.NullTime{Time: *effectiveStart, Valid: true}
		}
		if effectiveEnd != nil {
			sqlEffectiveEnd = sql.NullTime{Time: *effectiveEnd, Valid: true}
		}

		_, err := stmt.Exec(
			route.RouteCode, route.Origin, route.Destination, route.DepartureFix, route.RouteString,
			route.DepartureARTCC, route.ArrivalARTCC, route.TraversedARTCCs,
			route.CoordinationRequired, route.NavEqp, route.AssociatedPlay,
			sqlEffectiveStart, sqlEffectiveEnd, sourceFile,
		)
		if err != nil {
			// Log the problematic route for debugging
			log.Printf("ERROR saving CDR route: %+v, Error: %v", route, err)
			// Continue to try and save other routes, or return the error to stop all
			// For bulk load, sometimes it's better to log and continue, then review errors.
			// For now, let's return the error to be safe.
			return fmt.Errorf("failed to execute CDR insert for route_code '%s', origin '%s', dest '%s': %w", route.RouteCode, route.Origin, route.Destination, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for CDR routes: %w", err)
	}

	log.Printf("Successfully saved %d CDR routes from source: %s\n", len(routes), sourceFile)
	return nil
}

// SavePreferredRoutes (remains the same - already uses clear and load)
func SavePreferredRoutes(routes []models.PreferredRoute, sourceFile string, effectiveStart, effectiveEnd *time.Time) error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	if len(routes) == 0 {
		log.Println("No Preferred Routes provided to save.")
		return nil
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for preferred routes: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM nfdc_preferred_routes WHERE source_file = ?", sourceFile)
	if err != nil {
		return fmt.Errorf("failed to delete old preferred routes for source %s: %w", sourceFile, err)
	}
	log.Printf("Cleared existing preferred routes for source: %s\n", sourceFile)

	stmt, err := tx.Prepare(`
		INSERT INTO nfdc_preferred_routes (
			origin, route_string, destination, hours1, hours2, hours3,
			route_type, area, altitude, aircraft, direction, sequence,
			departure_artcc, arrival_artcc,
			effective_date_start, effective_date_end, source_file, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
	`) 
	if err != nil {
		return fmt.Errorf("failed to prepare Preferred Route insert statement: %w", err)
	}
	defer stmt.Close()

	for _, route := range routes {
		var sqlEffectiveStart, sqlEffectiveEnd sql.NullTime
		if effectiveStart != nil {
			sqlEffectiveStart = sql.NullTime{Time: *effectiveStart, Valid: true}
		}
		if effectiveEnd != nil {
			sqlEffectiveEnd = sql.NullTime{Time: *effectiveEnd, Valid: true}
		}

		_, err := stmt.Exec(
			route.Origin, route.RouteString, route.Destination, route.Hours1, route.Hours2, route.Hours3,
			route.Type, route.Area, route.Altitude, route.Aircraft, route.Direction, route.Sequence,
			route.DepartureARTCC, route.ArrivalARTCC,
			sqlEffectiveStart, sqlEffectiveEnd, sourceFile,
		)
		if err != nil {
			log.Printf("ERROR saving Preferred route: %+v, Error: %v", route, err)
			return fmt.Errorf("failed to execute Preferred Route insert for origin '%s', dest '%s': %w", route.Origin, route.Destination, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for Preferred Routes: %w", err)
	}

	log.Printf("Successfully saved %d Preferred Routes from source: %s\n", len(routes), sourceFile)
	return nil
}


// GetMaxEffectiveEndDateForSource (remains the same)
func GetMaxEffectiveEndDateForSource(sourceIdentifier string) (*time.Time, error) {
	if DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}
	var query string
	var pattern string

	if strings.EqualFold(sourceIdentifier, "CDR") {
		query = "SELECT MAX(effective_date_end) FROM cdm_routes WHERE source_file LIKE ?"
		pattern = "codedswap_db.csv%" 
	} else if strings.EqualFold(sourceIdentifier, "PreferredRoutes") {
		query = "SELECT MAX(effective_date_end) FROM nfdc_preferred_routes WHERE source_file LIKE ?"
		pattern = "prefroutes_db.csv%" 
	} else {
		return nil, fmt.Errorf("unknown source identifier for GetMaxEffectiveEndDateForSource: %s", sourceIdentifier)
	}
	
	var nullableDate sql.NullTime
	err := DB.QueryRow(query, pattern).Scan(&nullableDate)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No existing effective end date found in DB for source pattern: %s", pattern)
			return nil, nil 
		}
		return nil, fmt.Errorf("failed to query max effective_date_end for source pattern %s: %w", pattern, err)
	}

	if nullableDate.Valid {
		log.Printf("Found max effective_date_end %v for source pattern %s", nullableDate.Time.Format("2006-01-02"), pattern)
		return &nullableDate.Time, nil
	}
	log.Printf("No valid effective end date found in DB for source pattern (NULL value): %s", pattern)
	return nil, nil
}

// GetCdrRoutesForOD (remains the same)
func GetCdrRoutesForOD(origin, destination string, queryDate time.Time) ([]models.CdrRoute, error) {
	if DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}
	queryDateStr := queryDate.Format("2006-01-02")
	rows, err := DB.Query(`
		SELECT id, route_code, origin, destination, departure_fix, route_string,
		       departure_artcc, arrival_artcc, traversed_artccs,
		       coordination_required, nav_eqp, associated_play,
		       effective_date_start, effective_date_end, source_file, created_at, updated_at
		FROM cdm_routes
		WHERE origin = ? AND destination = ?
		  AND (effective_date_start IS NULL OR effective_date_start <= ?)
		  AND (effective_date_end IS NULL OR effective_date_end >= ?)
		ORDER BY route_code
	`, origin, destination, queryDateStr, queryDateStr)

	if err != nil {
		return nil, fmt.Errorf("failed to query CDR routes for OD %s-%s on %s: %w", origin, destination, queryDateStr, err)
	}
	defer rows.Close()
	var routes []models.CdrRoute
	for rows.Next() {
		var r models.CdrRoute
		var effStart, effEnd sql.NullTime 
		err := rows.Scan(
			&r.ID, &r.RouteCode, &r.Origin, &r.Destination, &r.DepartureFix, &r.RouteString,
			&r.DepartureARTCC, &r.ArrivalARTCC, &r.TraversedARTCCs,
			&r.CoordinationRequired, &r.NavEqp, &r.AssociatedPlay,
			&effStart, &effEnd, &r.SourceFile, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			log.Printf("ERROR: Failed to scan CDR route row: %v", err)
			continue 
		}
		if effStart.Valid {
			r.EffectiveDateStart = &effStart.Time
		}
		if effEnd.Valid {
			r.EffectiveDateEnd = &effEnd.Time
		}
		routes = append(routes, r)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating CDR route rows: %w", err)
	}
	log.Printf("Retrieved %d CDR routes for OD %s-%s on %s.\n", len(routes), origin, destination, queryDateStr)
	return routes, nil
}

// GetPreferredRoutesForOD (remains the same)
func GetPreferredRoutesForOD(origin, destination string, queryDate time.Time) ([]models.PreferredRoute, error) {
	if DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}
	queryDateStr := queryDate.Format("2006-01-02")
	rows, err := DB.Query(`
		SELECT id, origin, route_string, destination, hours1, hours2, hours3,
		       route_type, area, altitude, aircraft, direction, sequence,
		       departure_artcc, arrival_artcc,
		       effective_date_start, effective_date_end, source_file, created_at, updated_at
		FROM nfdc_preferred_routes
		WHERE origin = ? AND destination = ?
		  AND (effective_date_start IS NULL OR effective_date_start <= ?)
		  AND (effective_date_end IS NULL OR effective_date_end >= ?)
		ORDER BY sequence, route_type 
	`, origin, destination, queryDateStr, queryDateStr)

	if err != nil {
		return nil, fmt.Errorf("failed to query Preferred Routes for OD %s-%s on %s: %w", origin, destination, queryDateStr, err)
	}
	defer rows.Close()
	var routes []models.PreferredRoute
	for rows.Next() {
		var pr models.PreferredRoute
		var effStart, effEnd sql.NullTime
		err := rows.Scan(
			&pr.ID, &pr.Origin, &pr.RouteString, &pr.Destination, &pr.Hours1, &pr.Hours2, &pr.Hours3,
			&pr.Type, &pr.Area, &pr.Altitude, &pr.Aircraft, &pr.Direction, &pr.Sequence,
			&pr.DepartureARTCC, &pr.ArrivalARTCC,
			&effStart, &effEnd, &pr.SourceFile, &pr.CreatedAt, &pr.UpdatedAt,
		)
		if err != nil {
			log.Printf("ERROR: Failed to scan Preferred Route row: %v", err)
			continue
		}
		if effStart.Valid {
			pr.EffectiveDateStart = &effStart.Time
		}
		if effEnd.Valid {
			pr.EffectiveDateEnd = &effEnd.Time
		}
		routes = append(routes, pr)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating Preferred Route rows: %w", err)
	}
	log.Printf("Retrieved %d Preferred Routes for OD %s-%s on %s.\n", len(routes), origin, destination, queryDateStr)
	return routes, nil
}