// backend/models/meta.go (or add to route.go or advisory.go)
package models

import "time"

// DataSourceVersion tracks the freshness and metadata of downloaded static data sources.
type DataSourceVersion struct {
	ID                           int        `db:"id" json:"id"`
	SourceName                   string     `db:"source_name" json:"source_name"` // e.g., "CDR_CSV", "PREFERRED_ROUTES_CSV"
	SourceFileURL                string     `db:"source_file_url" json:"source_file_url"`
	LastDownloadedFilename       string     `db:"last_downloaded_filename" json:"last_downloaded_filename,omitempty"`
	EffectiveFrom                *time.Time `db:"effective_from" json:"effective_from,omitempty"`
	EffectiveUntil               *time.Time `db:"effective_until" json:"effective_until,omitempty"`
	LastCheckedOnFAASite         *time.Time `db:"last_checked_on_faa_site" json:"last_checked_on_faa_site,omitempty"` // When FAA page for effective dates was scraped
	LastSuccessfullyDownloadedAt *time.Time `db:"last_successfully_downloaded_at" json:"last_successfully_downloaded_at,omitempty"`
	DataHash                     string     `db:"data_hash" json:"data_hash,omitempty"` // Optional MD5/SHA256 of file
	CreatedAt                    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt                    time.Time  `db:"updated_at" json:"updated_at"`
}