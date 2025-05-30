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
// It uses an INSERT ... ON DUPLICATE KEY UPDATE strategy based on route_code.
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
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Note: Ensure your cdm_routes table has a UNIQUE constraint on `route_code`
	// for `ON DUPLICATE KEY UPDATE` to work as expected.
	// We defined `UNIQUE KEY uk_cdm_route_code (route_code)` in schema.sql.
	stmt, err := tx.Prepare(`
		INSERT INTO cdm_routes (
			route_code, origin, destination, departure_fix, route_string,
			departure_artcc, arrival_artcc, traversed_artccs,
			coordination_required, nav_eqp, associated_play,
			effective_date_start, effective_date_end, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			origin = VALUES(origin),
			destination = VALUES(destination),
			departure_fix = VALUES(departure_fix),
			route_string = VALUES(route_string),
			departure_artcc = VALUES(departure_artcc),
			arrival_artcc = VALUES(arrival_artcc),
			traversed_artccs = VALUES(traversed_artccs),
			coordination_required = VALUES(coordination_required),
			nav_eqp = VALUES(nav_eqp),
			associated_play = VALUES(associated_play),
			effective_date_start = VALUES(effective_date_start),
			effective_date_end = VALUES(effective_date_end),
			source_file = VALUES(source_file),
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare CDR insert statement: %w", err)
	}
	defer stmt.Close()

	for _, route := range routes {
		_, err := stmt.Exec(
			route.RouteCode, route.Origin, route.Destination, route.DepartureFix, route.RouteString,
			route.DepartureARTCC, route.ArrivalARTCC, route.TraversedARTCCs,
			route.CoordinationRequired, route.NavEqp, route.AssociatedPlay,
			effectiveStart, effectiveEnd, sourceFile,
		)
		if err != nil {
			// Consider collecting errors and continuing, or failing fast
			return fmt.Errorf("failed to execute CDR insert for route_code %s: %w", route.RouteCode, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for CDR routes: %w", err)
	}

	log.Printf("Successfully saved/updated %d CDR routes from source: %s\n", len(routes), sourceFile)
	return nil
}

// SavePreferredRoutes saves a slice of PreferredRoute objects to the database.
// This implementation uses a "clear and load" strategy for a given sourceFile
// or effective date range to simplify handling updates for this dataset.
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

	// Step 1: Delete existing routes for this sourceFile or effective period to avoid stale data.
	// This makes the operation idempotent for a given file/period.
	// Adjust the deletion criteria as needed. Using sourceFile is a simple way.
	// If effectiveStart and effectiveEnd are reliably from the FAA site for the *file*, use them.
	deleteQuery := "DELETE FROM nfdc_preferred_routes WHERE source_file = ?"
	var deleteArgs []interface{}
	deleteArgs = append(deleteArgs, sourceFile)

	// Alternatively, if you manage effective dates strictly:
	// deleteQuery = "DELETE FROM nfdc_preferred_routes WHERE effective_date_start = ? AND effective_date_end = ?"
	// deleteArgs = append(deleteArgs, effectiveStart, effectiveEnd)
	
	_, err = tx.Exec(deleteQuery, deleteArgs...)
	if err != nil {
		return fmt.Errorf("failed to delete old preferred routes for source %s: %w", sourceFile, err)
	}
	log.Printf("Cleared existing preferred routes for source: %s\n", sourceFile)


	// Step 2: Insert new routes
	stmt, err := tx.Prepare(`
		INSERT INTO nfdc_preferred_routes (
			origin, route_string, destination, hours1, hours2, hours3,
			route_type, area, altitude, aircraft, direction, sequence,
			departure_artcc, arrival_artcc,
			effective_date_start, effective_date_end, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare Preferred Route insert statement: %w", err)
	}
	defer stmt.Close()

	for _, route := range routes {
		_, err := stmt.Exec(
			route.Origin, route.RouteString, route.Destination, route.Hours1, route.Hours2, route.Hours3,
			route.Type, route.Area, route.Altitude, route.Aircraft, route.Direction, route.Sequence,
			route.DepartureARTCC, route.ArrivalARTCC,
			effectiveStart, effectiveEnd, sourceFile,
		)
		if err != nil {
			return fmt.Errorf("failed to execute Preferred Route insert for origin %s, dest %s: %w", route.Origin, route.Destination, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for Preferred Routes: %w", err)
	}

	log.Printf("Successfully saved %d Preferred Routes from source: %s\n", len(routes), sourceFile)
	return nil
}

// --- Query Functions (to be implemented later) ---

// GetCdrRoutesForOD retrieves active CDRs for a given origin and destination.
// func GetCdrRoutesForOD(origin, destination string, queryDate time.Time) ([]models.CdrRoute, error) {
// 	// SQL to select CDRs where origin=?, destination=? AND queryDate BETWEEN effective_date_start AND effective_date_end
// 	return nil, fmt.Errorf("GetCdrRoutesForOD not yet implemented")
// }

// GetPreferredRoutesForOD retrieves active Preferred Routes for a given origin and destination.
// func GetPreferredRoutesForOD(origin, destination string, queryDate time.Time) ([]models.PreferredRoute, error) {
// 	// SQL to select Preferred Routes where origin=?, destination=? AND queryDate BETWEEN effective_date_start AND effective_date_end
// 	return nil, fmt.Errorf("GetPreferredRoutesForOD not yet implemented")
// }