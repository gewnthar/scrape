// backend/database/advisory_store.go
package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
)

// SaveAdvisorySummaries saves a slice of AdvisorySummary objects to the database.
// It uses INSERT ... ON DUPLICATE KEY UPDATE based on summary_unique_key.
func SaveAdvisorySummaries(summaries []models.AdvisorySummary) error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	if len(summaries) == 0 {
		log.Println("No advisory summaries provided to save.")
		return nil
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for advisory summaries: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO advisory_summaries (
			advisory_date, summary_unique_key, list_page_raw_text, 
			issue_time_on_list_page, detail_page_params_json, 
			has_full_detail_saved, first_seen_at, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			list_page_raw_text = VALUES(list_page_raw_text),
			issue_time_on_list_page = VALUES(issue_time_on_list_page),
			detail_page_params_json = VALUES(detail_page_params_json),
			-- has_full_detail_saved should only be updated when a detail is actually saved/deleted
			last_seen_at = NOW()
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare advisory summary insert statement: %w", err)
	}
	defer stmt.Close()

	savedCount := 0
	for _, summary := range summaries {
		var issueTime sql.NullTime
		if summary.IssueTimeOnListPage != nil {
			issueTime.Time = *summary.IssueTimeOnListPage
			issueTime.Valid = true
		}

		// Ensure DetailPageParamsJSON is marshalled if DetailPageParams map is populated
		if summary.DetailPageParamsJSON == "" && summary.DetailPageParams != nil {
			paramsJSON, marshalErr := json.Marshal(summary.DetailPageParams)
			if marshalErr != nil {
				log.Printf("WARN: Could not marshal DetailPageParams for key %s: %v. Storing as empty JSON.", summary.SummaryUniqueKey, marshalErr)
				summary.DetailPageParamsJSON = "{}"
			} else {
				summary.DetailPageParamsJSON = string(paramsJSON)
			}
		}


		_, err := stmt.Exec(
			summary.AdvisoryDate.Format("2006-01-02"), // Ensure date is in YYYY-MM-DD format for SQL DATE
			summary.SummaryUniqueKey,
			summary.ListPageRawText,
			issueTime,
			summary.DetailPageParamsJSON,
			summary.HasFullDetailSaved, // This will typically be false on initial summary save
			// first_seen_at and last_seen_at handled by SQL (NOW() or ON DUPLICATE KEY UPDATE)
		)
		if err != nil {
			// It's possible to collect errors and continue, or fail fast.
			// For now, fail fast within the transaction.
			log.Printf("ERROR: Failed to execute advisory summary insert for key %s: %v", summary.SummaryUniqueKey, err)
			return fmt.Errorf("failed to execute advisory summary insert for key %s: %w", summary.SummaryUniqueKey, err)
		}
		savedCount++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for advisory summaries: %w", err)
	}

	log.Printf("Successfully saved/updated %d advisory summaries.\n", savedCount)
	return nil
}

// SaveAdvisoryDetail saves a single AdvisoryDetail object and updates the corresponding summary.
func SaveAdvisoryDetail(detail models.AdvisoryDetail) error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for advisory detail: %w", err)
	}
	defer tx.Rollback()

	// Marshal ParsedAffectedFacilities if the slice is populated
	if detail.ParsedAffectedFacilitiesJSON == "" && detail.ParsedAffectedFacilities != nil {
		facilitiesJSON, marshalErr := json.Marshal(detail.ParsedAffectedFacilities)
		if marshalErr != nil {
			log.Printf("WARN: Could not marshal ParsedAffectedFacilities for summary key %s: %v. Storing as null or empty JSON.", detail.SummaryKey, marshalErr)
			// Decide on fallback, e.g., "[]" or let it be NULL if schema allows
			detail.ParsedAffectedFacilitiesJSON = "[]" 
		} else {
			detail.ParsedAffectedFacilitiesJSON = string(facilitiesJSON)
		}
	}
	
	var parsedRemarks sql.NullString
	if detail.ParsedRemarks != nil {
		parsedRemarks.String = *detail.ParsedRemarks
		parsedRemarks.Valid = true
	}
	var parsedStart, parsedEnd sql.NullTime
	if detail.ParsedEventTimeStartZulu != nil {
		parsedStart.Time = *detail.ParsedEventTimeStartZulu
		parsedStart.Valid = true
	}
	if detail.ParsedEventTimeEndZulu != nil {
		parsedEnd.Time = *detail.ParsedEventTimeEndZulu
		parsedEnd.Valid = true
	}


	// Insert or Update the detail. Using summary_key as unique for upsert.
	_, err = tx.Exec(`
		INSERT INTO advisory_details (
			summary_key, source, full_text_content, 
			parsed_event_time_start_zulu, parsed_event_time_end_zulu, 
			parsed_affected_facilities_json, parsed_remarks, fetched_and_saved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			source = VALUES(source),
			full_text_content = VALUES(full_text_content),
			parsed_event_time_start_zulu = VALUES(parsed_event_time_start_zulu),
			parsed_event_time_end_zulu = VALUES(parsed_event_time_end_zulu),
			parsed_affected_facilities_json = VALUES(parsed_affected_facilities_json),
			parsed_remarks = VALUES(parsed_remarks),
			fetched_and_saved_at = NOW()
	`, detail.SummaryKey, detail.Source, detail.FullTextContent, parsedStart, parsedEnd, detail.ParsedAffectedFacilitiesJSON, parsedRemarks)

	if err != nil {
		return fmt.Errorf("failed to save advisory detail for summary_key %s: %w", detail.SummaryKey, err)
	}

	// Update the corresponding advisory_summary to mark has_full_detail_saved = TRUE
	_, err = tx.Exec(`
		UPDATE advisory_summaries 
		SET has_full_detail_saved = TRUE, last_seen_at = NOW()
		WHERE summary_unique_key = ?
	`, detail.SummaryKey)

	if err != nil {
		return fmt.Errorf("failed to update advisory_summary flag for summary_key %s: %w", detail.SummaryKey, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for advisory detail: %w", err)
	}

	log.Printf("Successfully saved advisory detail for summary_key: %s and updated summary flag.\n", detail.SummaryKey)
	return nil
}


// GetAdvisorySummariesByDate retrieves all advisory summaries for a specific date.
func GetAdvisorySummariesByDate(queryDate time.Time) ([]models.AdvisorySummary, error) {
	if DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	rows, err := DB.Query(`
		SELECT id, advisory_date, summary_unique_key, list_page_raw_text, 
		       issue_time_on_list_page, detail_page_params_json, 
		       has_full_detail_saved, first_seen_at, last_seen_at
		FROM advisory_summaries 
		WHERE advisory_date = ?
		ORDER BY issue_time_on_list_page DESC, id DESC
	`, queryDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query advisory summaries for date %s: %w", queryDate.Format("2006-01-02"), err)
	}
	defer rows.Close()

	var summaries []models.AdvisorySummary
	for rows.Next() {
		var s models.AdvisorySummary
		var issueTime sql.NullTime // Handle nullable TIME from DB
		// detail_page_params_json is read as string, then unmarshalled if needed
		
		err := rows.Scan(
			&s.ID, &s.AdvisoryDate, &s.SummaryUniqueKey, &s.ListPageRawText,
			&issueTime, &s.DetailPageParamsJSON,
			&s.HasFullDetailSaved, &s.FirstSeenAt, &s.LastSeenAt,
		)
		if err != nil {
			log.Printf("ERROR: Failed to scan advisory summary row: %v", err)
			// Decide if to continue or return error for the whole batch
			continue 
		}
		if issueTime.Valid {
			s.IssueTimeOnListPage = &issueTime.Time
		}
		// Optionally unmarshal JSON params here if always needed, or do it in service layer
		if s.DetailPageParamsJSON != "" {
			var params map[string]string
			if unmarshalErr := json.Unmarshal([]byte(s.DetailPageParamsJSON), &params); unmarshalErr == nil {
				s.DetailPageParams = params
			} else {
				log.Printf("WARN: Could not unmarshal DetailPageParamsJSON for summary %s: %v", s.SummaryUniqueKey, unmarshalErr)
			}
		}
		summaries = append(summaries, s)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating advisory summary rows: %w", err)
	}

	log.Printf("Retrieved %d advisory summaries for date %s.\n", len(summaries), queryDate.Format("2006-01-02"))
	return summaries, nil
}

// GetAdvisoryDetail retrieves a specific advisory detail by its summary_key.
func GetAdvisoryDetail(summaryKey string) (*models.AdvisoryDetail, error) {
	if DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}
	
	var detail models.AdvisoryDetail
	var parsedStart, parsedEnd sql.NullTime
	var parsedRemarks sql.NullString
	// parsed_affected_facilities_json is read as string, then unmarshalled if needed

	row := DB.QueryRow(`
		SELECT id, summary_key, source, full_text_content, 
		       parsed_event_time_start_zulu, parsed_event_time_end_zulu,
		       parsed_affected_facilities_json, parsed_remarks, fetched_and_saved_at
		FROM advisory_details
		WHERE summary_key = ?
	`, summaryKey)

	err := row.Scan(
		&detail.ID, &detail.SummaryKey, &detail.Source, &detail.FullTextContent,
		&parsedStart, &parsedEnd,
		&detail.ParsedAffectedFacilitiesJSON, &parsedRemarks, &detail.FetchedAndSavedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error, just no result
		}
		return nil, fmt.Errorf("failed to query advisory detail for summary_key %s: %w", summaryKey, err)
	}

	if parsedStart.Valid {
		detail.ParsedEventTimeStartZulu = &parsedStart.Time
	}
	if parsedEnd.Valid {
		detail.ParsedEventTimeEndZulu = &parsedEnd.Time
	}
	if parsedRemarks.Valid {
		detail.ParsedRemarks = &parsedRemarks.String
	}
	// Optionally unmarshal JSON facilities here, or do it in service layer
	if detail.ParsedAffectedFacilitiesJSON != "" {
		var facilities []string
		if unmarshalErr := json.Unmarshal([]byte(detail.ParsedAffectedFacilitiesJSON), &facilities); unmarshalErr == nil {
			detail.ParsedAffectedFacilities = facilities
		} else {
			log.Printf("WARN: Could not unmarshal ParsedAffectedFacilitiesJSON for detail %s: %v", detail.SummaryKey, unmarshalErr)
		}
	}


	log.Printf("Retrieved advisory detail for summary_key: %s.\n", summaryKey)
	return &detail, nil
}

// MarkAdvisoryDetailAsSaved updates the advisory_summaries table.
// This is now integrated into SaveAdvisoryDetail within a transaction.
// Kept here as a comment if a separate function is ever needed.
/*
func MarkAdvisoryDetailAsSaved(summaryKey string, saved bool) error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	_, err := DB.Exec(`
		UPDATE advisory_summaries
		SET has_full_detail_saved = ?, last_seen_at = NOW()
		WHERE summary_unique_key = ?
	`, saved, summaryKey)
	if err != nil {
		return fmt.Errorf("failed to update has_full_detail_saved for summary_key %s: %w", summaryKey, err)
	}
	log.Printf("Marked advisory summary %s as detail saved: %t\n", summaryKey, saved)
	return nil
}
*/