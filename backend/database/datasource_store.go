// backend/database/datasource_store.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
)

// LogDataSourceVersionUpdate inserts or updates a record in the data_source_versions table.
// This indicates when a static data source (like a CSV) was last checked, downloaded,
// and what its purported effective dates are.
func LogDataSourceVersionUpdate(
	sourceName string,
	sourceURL string,
	downloadedFilename string,
	effectiveFrom *time.Time,
	effectiveUntil *time.Time,
	lastCheckedOnFAASite *time.Time, // When the FAA page for effective dates was last scraped
	lastSuccessfullyDownloadedAt *time.Time,
	dataHash sql.NullString, // Optional hash of the file content
) error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	var sqlEffectiveFrom, sqlEffectiveUntil, sqlLastChecked, sqlLastDownloaded sql.NullTime

	if effectiveFrom != nil {
		sqlEffectiveFrom = sql.NullTime{Time: *effectiveFrom, Valid: true}
	}
	if effectiveUntil != nil {
		sqlEffectiveUntil = sql.NullTime{Time: *effectiveUntil, Valid: true}
	}
	if lastCheckedOnFAASite != nil {
		sqlLastChecked = sql.NullTime{Time: *lastCheckedOnFAASite, Valid: true}
	}
	if lastSuccessfullyDownloadedAt != nil {
		sqlLastDownloaded = sql.NullTime{Time: *lastSuccessfullyDownloadedAt, Valid: true}
	}
	

	query := `
		INSERT INTO data_source_versions (
			source_name, source_file_url, last_downloaded_filename, 
			effective_from, effective_until, last_checked_on_faa_site, 
			last_successfully_downloaded_at, data_hash, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			source_file_url = VALUES(source_file_url),
			last_downloaded_filename = VALUES(last_downloaded_filename),
			effective_from = VALUES(effective_from),
			effective_until = VALUES(effective_until),
			last_checked_on_faa_site = VALUES(last_checked_on_faa_site),
			last_successfully_downloaded_at = VALUES(last_successfully_downloaded_at),
			data_hash = VALUES(data_hash),
			updated_at = NOW()
	`

	_, err := DB.Exec(query,
		sourceName, sourceURL, downloadedFilename,
		sqlEffectiveFrom, sqlEffectiveUntil, sqlLastChecked,
		sqlLastDownloaded, dataHash,
	)

	if err != nil {
		log.Printf("ERROR Database: Failed to log/update data source version for '%s': %v", sourceName, err)
		return fmt.Errorf("failed to log data source version for %s: %w", sourceName, err)
	}

	log.Printf("Database: Successfully logged/updated data source version for '%s'. Effective Until: %v, Downloaded: %v\n",
		sourceName, effectiveUntil, lastSuccessfullyDownloadedAt)
	return nil
}

// GetDataSourceVersions retrieves all records from the data_source_versions table.
func GetDataSourceVersions() ([]models.DataSourceVersion, error) { // Assuming models.DataSourceVersion struct
	if DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	rows, err := DB.Query(`
		SELECT id, source_name, source_file_url, last_downloaded_filename,
		       effective_from, effective_until, last_checked_on_faa_site,
		       last_successfully_downloaded_at, data_hash, created_at, updated_at
		FROM data_source_versions
		ORDER BY source_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query data_source_versions: %w", err)
	}
	defer rows.Close()

	var versions []models.DataSourceVersion
	for rows.Next() {
		var v models.DataSourceVersion
		var effFrom, effUntil, lastChecked, lastDownloaded sql.NullTime
		var lastDownloadedFilename, dataHash sql.NullString

		err := rows.Scan(
			&v.ID, &v.SourceName, &v.SourceFileURL, &lastDownloadedFilename,
			&effFrom, &effUntil, &lastChecked,
			&lastDownloaded, &dataHash, &v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			log.Printf("ERROR Database: Failed to scan data_source_version row: %v", err)
			continue
		}
		if lastDownloadedFilename.Valid {
			v.LastDownloadedFilename = lastDownloadedFilename.String
		}
		if effFrom.Valid {
			v.EffectiveFrom = &effFrom.Time
		}
		if effUntil.Valid {
			v.EffectiveUntil = &effUntil.Time
		}
		if lastChecked.Valid {
			v.LastCheckedOnFAASite = &lastChecked.Time
		}
		if lastDownloaded.Valid {
			v.LastSuccessfullyDownloadedAt = &lastDownloaded.Time
		}
		if dataHash.Valid {
			v.DataHash = dataHash.String
		}
		versions = append(versions, v)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating data_source_version rows: %w", err)
	}
	return versions, nil
}