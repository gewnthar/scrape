 -- database_schemas/schema.sql

-- Ensure you are connected to your target database, e.g., by running:
-- USE faa_dst_db; 
-- before executing this script, if running manually.

-- Static CDR Data (from codedswap_db.csv)
CREATE TABLE IF NOT EXISTS cdm_routes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    
    route_code VARCHAR(20) NOT NULL,  -- e.g., "JFKMIA48K"
    origin VARCHAR(10) NOT NULL,
    destination VARCHAR(10) NOT NULL,
    departure_fix VARCHAR(50) NULL,
    route_string TEXT NULL,
    departure_artcc VARCHAR(10) NULL,
    arrival_artcc VARCHAR(10) NULL,
    traversed_artccs VARCHAR(255) NULL,
    coordination_required VARCHAR(5) NULL, -- Stores "Y", "N", or other codes
    nav_eqp VARCHAR(10) NULL,
    associated_play VARCHAR(255) NULL,

    effective_date_start DATE NULL, -- Effective date of the CSV file data (from FAA page)
    effective_date_end DATE NULL,   -- Effective date of the CSV file data (from FAA page)
    source_file VARCHAR(255) NULL,  -- Name of the CSV file it came from (e.g., codedswap_db.csv_YYYY-MM-DD)

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_cdm_route_code (route_code) 
    -- Consider if a composite unique key is more appropriate if route_code alone isn't always unique
    -- e.g. UNIQUE KEY uk_cdm_origin_dest_code (origin, destination, route_code)
);

-- Static Preferred Routes Data (from prefroutes_db.csv)
CREATE TABLE IF NOT EXISTS nfdc_preferred_routes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    
    origin VARCHAR(10) NOT NULL,
    route_string TEXT NULL,
    destination VARCHAR(10) NOT NULL,
    hours1 VARCHAR(255) NULL,
    hours2 VARCHAR(255) NULL,
    hours3 VARCHAR(255) NULL,
    route_type VARCHAR(20) NULL, 
    area VARCHAR(255) NULL,
    altitude VARCHAR(255) NULL,
    aircraft TEXT NULL, 
    direction VARCHAR(100) NULL,
    sequence VARCHAR(50) NULL, 
    departure_artcc VARCHAR(10) NULL,
    arrival_artcc VARCHAR(10) NULL,

    effective_date_start DATE NULL, -- Effective date of the CSV file data (from FAA page)
    effective_date_end DATE NULL,   -- Effective date of the CSV file data (from FAA page)
    source_file VARCHAR(255) NULL,  -- Name of the CSV file it came from (e.g., prefroutes_db.csv_YYYY-MM-DD)

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_nfdc_prefroutes_origin_dest (origin, destination),
    INDEX idx_nfdc_prefroutes_effective_dates (effective_date_start, effective_date_end)
    -- A UNIQUE KEY for preferred routes might be composite, e.g.:
    -- UNIQUE KEY uk_nfdc_pref_route (origin, destination, route_string(255), sequence, route_type) 
    -- (Using route_string(255) because TEXT columns can't be fully unique indexed without length)
    -- This needs careful consideration based on data uniqueness. For now, primary ID is the main uniqueness.
);

-- Dynamic Advisory Summaries (from FAA advisory list page)
CREATE TABLE IF NOT EXISTS advisory_summaries (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    advisory_date DATE NOT NULL,                 -- The primary Zulu date of the advisory
    
    -- Unique key for a summary on a given day.
    -- Constructed in Go from stable parts of the list item text/links.
    summary_unique_key VARCHAR(768) UNIQUE NOT NULL, 
    
    list_page_raw_text TEXT NOT NULL,            -- The full text line from the FAA advisory list
    issue_time_on_list_page TIME NULL,           -- Extracted time like 17:31Z (if available from list)
    
    -- JSON object storing parameters needed to fetch the detail page for this advisory
    detail_page_params_json JSON NOT NULL,       -- e.g., {"advn": "57", "adv_date": "05292025", ...}
    
    has_full_detail_saved BOOLEAN NOT NULL DEFAULT FALSE, -- True if corresponding entry in advisory_details exists
    
    first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,      -- When this summary was first scraped
    last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, -- If we re-scrape list pages and see it again

    INDEX idx_advisory_summaries_advisory_date (advisory_date),
    INDEX idx_advisory_summaries_has_detail (has_full_detail_saved)
);

-- Dynamic Advisory Details (full text and parsed elements of specific advisories)
CREATE TABLE IF NOT EXISTS advisory_details (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    summary_key VARCHAR(768) UNIQUE NOT NULL, -- Must match an advisory_summaries.summary_unique_key

    source VARCHAR(50) NOT NULL DEFAULT 'FAA_ADVISORY', -- e.g., 'FAA_ADVISORY', 'RAT_READER'
    full_text_content MEDIUMTEXT NOT NULL,     
    
    -- Examples of structured data parsed from full_text_content
    parsed_event_time_start_zulu DATETIME NULL,
    parsed_event_time_end_zulu DATETIME NULL,
    parsed_affected_facilities JSON NULL,   -- Store as a JSON array of strings: ["ZDV", "DEN"]
    parsed_remarks TEXT NULL,               -- For storing lengthy remarks/modifications sections
    -- Add more parsed fields as you identify them (e.g., specific CDRs mentioned, flight types)
    
    fetched_and_saved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- When this detail was scraped and saved
    
    FOREIGN KEY (summary_key) REFERENCES advisory_summaries(summary_unique_key) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Table to track versions/effective dates of downloaded static data files (like CDRs, Preferred Routes)
CREATE TABLE IF NOT EXISTS data_source_versions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    source_name VARCHAR(100) UNIQUE NOT NULL, -- e.g., "CDR_CSV", "PREFERRED_ROUTES_CSV"
    
    source_file_url VARCHAR(512) NULL,        -- The URL from where the file is downloaded
    last_downloaded_filename VARCHAR(255) NULL, -- The actual filename downloaded (could include dates)
    
    -- Effective dates for the data within the last downloaded file, as scraped from FAA's website
    effective_from DATE NULL,
    effective_until DATE NULL,
    
    last_checked_on_faa_site_for_eff_date TIMESTAMP NULL, -- When we last scraped the FAA page for its effective dates
    last_successfully_downloaded_at TIMESTAMP NULL,    -- When the CSV file itself was last downloaded
    
    -- Optional: MD5 or SHA256 hash of the downloaded file to detect if content changed even if dates didn't
    data_hash VARCHAR(64) NULL, 
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);