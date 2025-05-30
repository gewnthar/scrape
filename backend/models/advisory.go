// backend/models/advisory.go
package models

import "time"

// AdvisorySummary represents an item from the FAA advisory list page.
type AdvisorySummary struct {
	ID                      int64             `db:"id"`
	AdvisoryDate            time.Time         `db:"advisory_date"`            // YYYY-MM-DD
	SummaryUniqueKey        string            `db:"summary_unique_key"`       // Must be constructed to be unique
	ListPageRawText         string            `db:"list_page_raw_text"`
	IssueTimeOnListPage     *time.Time        `db:"issue_time_on_list_page"`  // Nullable time (HH:MM:SS)
	DetailPageParamsJSON    string            `db:"detail_page_params_json"`  // Store as JSON string
	HasFullDetailSaved      bool              `db:"has_full_detail_saved"`
	FirstSeenAt             time.Time         `db:"first_seen_at"`
	LastSeenAt              time.Time         `db:"last_seen_at"`

	// Not in DB table, but useful for holding unmarshalled params
	DetailPageParams map[string]string `db:"-" json:"-"` 
}

// AdvisoryDetail represents the full content of a specific FAA advisory.
type AdvisoryDetail struct {
	ID                           int64      `db:"id"`
	SummaryKey                   string     `db:"summary_key"` // FK to AdvisorySummary.SummaryUniqueKey
	Source                       string     `db:"source"`      // e.g., "FAA_ADVISORY"
	FullTextContent              string     `db:"full_text_content"`
	ParsedEventTimeStartZulu     *time.Time `db:"parsed_event_time_start_zulu"` // Nullable
	ParsedEventTimeEndZulu       *time.Time `db:"parsed_event_time_end_zulu"`   // Nullable
	ParsedAffectedFacilitiesJSON string     `db:"parsed_affected_facilities_json"`// Store as JSON string
	ParsedRemarks                *string    `db:"parsed_remarks"` // Nullable
	FetchedAndSavedAt            time.Time  `db:"fetched_and_saved_at"`

	// Not in DB table, but useful for holding unmarshalled facilities
	ParsedAffectedFacilities []string `db:"-" json:"-"`
}