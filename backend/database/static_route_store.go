// backend/database/static_route_store.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings" // Keep for GetMaxEffectiveEndDateForSource
	"time"

	"github.com/gewnthar/scrape/backend/models"
)

// SaveCdrRoutes saves a slice of CdrRoute objects to the database.
// Uses a "clear and load" strategy for a given sourceFile.
func SaveCdrRoutes(routes []models.CdrRoute, sourceFile string, effectiveStart, effectiveEnd *time.Time) error {
	if DB == nil {
		return fmt.Errorf("DB store: Database connection is not initialized for SaveCdrRoutes")
	}
	if len(routes) == 0 {
		log.Println("DB store: No CDR routes provided to save.")
		return nil
	}

	log.Printf("DB store: Attempting to save %d CDR routes from source: %s\n", len(routes), sourceFile)

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("DB store: Failed to begin transaction for CDR routes: %w", err)
	}
	// Defer rollback. If Commit() is successful, Rollback() does nothing.
	// If Commit() is not called (due to error), Rollback() will execute.
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-panic after rollback
		} else if err != nil {
			log.Println("DB store: Rolling back CDR transaction due to error:", err)
			tx.Rollback()
		}
	}()


	// Step 1: Delete existing routes for this sourceFile.
	log.Printf("DB store: Clearing existing CDR routes for source_file: %s\n", sourceFile)
	res, err := tx.Exec("DELETE FROM cdm_routes WHERE source_file = ?", sourceFile)
	if err != nil {
		// err is already being set for the defer, no need to return here if we want to see if commit works/fails
		// return fmt.Errorf("failed to delete old CDR routes for source %s: %w", sourceFile, err)
		log.Printf("DB store: ERROR clearing old CDR routes for source %s: %v\n", sourceFile, err)
        // Let's try to commit anyway or see if the inserts fail
	} else {
		rowsAffected, _ := res.RowsAffected()
		log.Printf("DB store: Cleared %d old CDR routes for source_file: %s\n", rowsAffected, sourceFile)
	}


	// Step 2: Insert new routes
	stmt, err := tx.Prepare(`
		INSERT INTO cdm_routes (
			route_code, origin, destination, departure_fix, route_string,
			departure_artcc, arrival_artcc, traversed_artccs,
			coordination_required, nav_eqp, associated_play,
			effective_date_start, effective_date_end, source_file 
			-- created_at is default, updated_at is default/on update
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("DB store: Failed to prepare CDR insert statement: %w", err)
	}
	defer stmt.Close()

	var insertErrors []string
	for i, route := range routes {
		var sqlEffectiveStart, sqlEffectiveEnd sql.NullTime
		if effectiveStart != nil {
			sqlEffectiveStart = sql.NullTime{Time: *effectiveStart, Valid: true}
		}
		if effectiveEnd != nil {
			sqlEffectiveEnd = sql.NullTime{Time: *effectiveEnd, Valid: true}
		}

		// Diagnostic log for the data being inserted
		log.Printf("DB store (CDR): Attempting insert [%d/%d] - Origin: '%s', Dest: '%s', Code: '%s', Route: '%.30s...', SourceFile: '%s'\n",
			i+1, len(routes), route.Origin, route.Destination, route.RouteCode, route.RouteString, sourceFile)

		_, execErr := stmt.Exec(
			route.RouteCode, route.Origin, route.Destination, route.DepartureFix, route.RouteString,
			route.DepartureARTCC, route.ArrivalARTCC, route.TraversedARTCCs,
			route.CoordinationRequired, route.NavEqp, route.AssociatedPlay,
			sqlEffectiveStart, sqlEffectiveEnd, sourceFile,
		)
		if execErr != nil {
			errMsg := fmt.Sprintf("DB store: Failed to execute CDR insert for route_code '%s', origin '%s', dest '%s': %v", route.RouteCode, route.Origin, route.Destination, execErr)
			log.Println(errMsg)
			insertErrors = append(insertErrors, errMsg)
			// Optionally continue to try and insert other rows, or return immediately
			// For now, let's collect errors and try to commit what succeeded before the error
			// Or, to ensure atomicity, return immediately on first error:
			// err = execErr // set the outer err variable for defer
			// return fmt.Errorf(errMsg) 
		}
	}

    if len(insertErrors) > 0 {
        // If there were insert errors, the transaction will be rolled back by defer.
        // We set the main 'err' so defer knows about it.
        err = fmt.Errorf("DB store: Encountered %d errors during CDR batch insert. First error: %s", len(insertErrors), insertErrors[0])
        log.Println(err.Error()) // Log the summary error
        return err // This will trigger rollback
    }

	log.Println("DB store: CDR Insert loop completed. Attempting to commit transaction...")
	if err = tx.Commit(); err != nil { // Assign to outer err
		return fmt.Errorf("DB store: Failed to commit transaction for CDR routes: %w", err)
	}

	log.Printf("DB store: Successfully committed %d CDR routes from source: %s\n", len(routes)-len(insertErrors), sourceFile)
	return nil
}

// SavePreferredRoutes saves a slice of PreferredRoute objects to the database.
// Uses a "clear and load" strategy for a given sourceFile.
func SavePreferredRoutes(routes []models.PreferredRoute, sourceFile string, effectiveStart, effectiveEnd *time.Time) error {
	if DB == nil {
		return fmt.Errorf("DB store: Database connection is not initialized for SavePreferredRoutes")
	}
	if len(routes) == 0 {
		log.Println("DB store: No Preferred Routes provided to save.")
		return nil
	}
	log.Printf("DB store: Attempting to save %d Preferred Routes from source: %s\n", len(routes), sourceFile)


	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("DB store: Failed to begin transaction for preferred routes: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) 
		} else if err != nil {
			log.Println("DB store: Rolling back Preferred Routes transaction due to error:", err)
			tx.Rollback()
		}
	}()

	log.Printf("DB store: Clearing existing Preferred Routes for source_file: %s\n", sourceFile)
	res, err := tx.Exec("DELETE FROM nfdc_preferred_routes WHERE source_file = ?", sourceFile)
	if err != nil {
		log.Printf("DB store: ERROR clearing old Preferred Routes for source %s: %v\n", sourceFile, err)
	} else {
		rowsAffected, _ := res.RowsAffected()
		log.Printf("DB store: Cleared %d old Preferred Routes for source_file: %s\n", rowsAffected, sourceFile)
	}


	stmt, err := tx.Prepare(`
		INSERT INTO nfdc_preferred_routes (
			origin, route_string, destination, hours1, hours2, hours3,
			route_type, area, altitude, aircraft, direction, sequence,
			departure_artcc, arrival_artcc,
			effective_date_start, effective_date_end, source_file
			-- created_at is default, updated_at is default/on update
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("DB store: Failed to prepare Preferred Route insert statement: %w", err)
	}
	defer stmt.Close()

	var insertErrors []string
	for i, route := range routes {
		var sqlEffectiveStart, sqlEffectiveEnd sql.NullTime
		if effectiveStart != nil {
			sqlEffectiveStart = sql.NullTime{Time: *effectiveStart, Valid: true}
		}
		if effectiveEnd != nil {
			sqlEffectiveEnd = sql.NullTime{Time: *effectiveEnd, Valid: true}
		}
		
		log.Printf("DB store (PrefRoute): Attempting insert [%d/%d] - Origin: '%s', Dest: '%s', Type: '%s', Route: '%.30s...', SourceFile: '%s'\n",
			i+1, len(routes), route.Origin, route.Destination, route.Type, route.RouteString, sourceFile)

		_, execErr := stmt.Exec(
			route.Origin, route.RouteString, route.Destination, route.Hours1, route.Hours2, route.Hours3,
			route.Type, route.Area, route.Altitude, route.Aircraft, route.Direction, route.Sequence,
			route.DepartureARTCC, route.ArrivalARTCC,
			sqlEffectiveStart, sqlEffectiveEnd, sourceFile,
		)
		if execErr != nil {
			errMsg := fmt.Sprintf("DB store: Failed to execute Preferred Route insert for origin '%s', dest '%s': %v", route.Origin, route.Destination, execErr)
			log.Println(errMsg)
			insertErrors = append(insertErrors, errMsg)
			// err = execErr 
			// return fmt.Errorf(errMsg)
		}
	}

    if len(insertErrors) > 0 {
        err = fmt.Errorf("DB store: Encountered %d errors during Preferred Routes batch insert. First error: %s", len(insertErrors), insertErrors[0])
        log.Println(err.Error())
        return err // This will trigger rollback
    }

	log.Println("DB store: Preferred Routes Insert loop completed. Attempting to commit transaction...")
	if err = tx.Commit(); err != nil { // Assign to outer err
		return fmt.Errorf("DB store: Failed to commit transaction for Preferred Routes: %w", err)
	}

	log.Printf("DB store: Successfully committed %d Preferred Routes from source: %s\n", len(routes)-len(insertErrors), sourceFile)
	return nil
}


// GetMaxEffectiveEndDateForSource (remains the same as your latest version)
func GetMaxEffectiveEndDateForSource(sourceIdentifier string) (*time.Time, error) {
	if DB == nil {
		return nil, fmt.Errorf("DB store: database connection is not initialized for GetMaxEffectiveEndDateForSource")
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
		return nil, fmt.Errorf("DB store: unknown source identifier for GetMaxEffectiveEndDateForSource: %s", sourceIdentifier)
	}
	
	var nullableDate sql.NullTime
	dbErr := DB.QueryRow(query, pattern).Scan(&nullableDate) // Renamed err to dbErr to avoid conflict with outer 'err' in defer
	if dbErr != nil {
		if dbErr == sql.ErrNoRows {
			log.Printf("DB store: No existing effective end date found in DB for source pattern: %s", pattern)
			return nil, nil 
		}
		return nil, fmt.Errorf("DB store: failed to query max effective_date_end for source pattern %s: %w", pattern, dbErr)
	}

	if nullableDate.Valid {
		log.Printf("DB store: Found max effective_date_end %v for source pattern %s", nullableDate.Time.Format("2006-01-02"), pattern)
		return &nullableDate.Time, nil
	}
	log.Printf("DB store: No valid effective end date found in DB for source pattern (NULL value): %s", pattern)
	return nil, nil
}

// GetCdrRoutesForOD (remains the same as your latest version)
func GetCdrRoutesForOD(origin, destination string, queryDate time.Time) ([]models.CdrRoute, error) {
	if DB == nil {
		return nil, fmt.Errorf("DB store: database connection is not initialized for GetCdrRoutesForOD")
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
		return nil, fmt.Errorf("DB store: failed to query CDR routes for OD %s-%s on %s: %w", origin, destination, queryDateStr, err)
	}
	defer rows.Close()
	var routes []models.CdrRoute
	for rows.Next() {
		var r models.CdrRoute
		var effStart, effEnd sql.NullTime 
		scanErr := rows.Scan( // Renamed err to scanErr
			&r.ID, &r.RouteCode, &r.Origin, &r.Destination, &r.DepartureFix, &r.RouteString,
			&r.DepartureARTCC, &r.ArrivalARTCC, &r.TraversedARTCCs,
			&r.CoordinationRequired, &r.NavEqp, &r.AssociatedPlay,
			&effStart, &effEnd, &r.SourceFile, &r.CreatedAt, &r.UpdatedAt,
		)
		if scanErr != nil {
			log.Printf("DB store ERROR: Failed to scan CDR route row: %v", scanErr)
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
	if err = rows.Err(); err != nil { // Check for errors during iteration
		return nil, fmt.Errorf("DB store: error iterating CDR route rows: %w", err)
	}
	log.Printf("DB store: Retrieved %d CDR routes for OD %s-%s on %s.\n", len(routes), origin, destination, queryDateStr)
	return routes, nil
}

// GetPreferredRoutesForOD (remains the same as your latest version)
func GetPreferredRoutesForOD(origin, destination string, queryDate time.Time) ([]models.PreferredRoute, error) {
	if DB == nil {
		return nil, fmt.Errorf("DB store: database connection is not initialized for GetPreferredRoutesForOD")
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
		return nil, fmt.Errorf("DB store: failed to query Preferred Routes for OD %s-%s on %s: %w", origin, destination, queryDateStr, err)
	}
	defer rows.Close()
	var routes []models.PreferredRoute
	for rows.Next() {
		var pr models.PreferredRoute
		var effStart, effEnd sql.NullTime
		scanErr := rows.Scan( // Renamed err to scanErr
			&pr.ID, &pr.Origin, &pr.RouteString, &pr.Destination, &pr.Hours1, &pr.Hours2, &pr.Hours3,
			&pr.Type, &pr.Area, &pr.Altitude, &pr.Aircraft, &pr.Direction, &pr.Sequence,
			&pr.DepartureARTCC, &pr.ArrivalARTCC,
			&effStart, &effEnd, &pr.SourceFile, &pr.CreatedAt, &pr.UpdatedAt,
		)
		if scanErr != nil {
			log.Printf("DB store ERROR: Failed to scan Preferred Route row: %v", scanErr)
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
	if err = rows.Err(); err != nil { // Check for errors during iteration
		return nil, fmt.Errorf("DB store: error iterating Preferred Route rows: %w", err)
	}
	log.Printf("DB store: Retrieved %d Preferred Routes for OD %s-%s on %s.\n", len(routes), origin, destination, queryDateStr)
	return routes, nil
}