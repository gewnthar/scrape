-- database_schemas/schema.sql

-- Ensure you are connected to your target database, e.g., USE faa_dst_db;

-- Static CDR Data (from codedswap_db.csv)
DROP TABLE IF EXISTS cdm_routes; -- Drop if exists to easily apply change to key
CREATE TABLE IF NOT EXISTS cdm_routes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    
    route_code VARCHAR(20) NULL,  -- Allow NULL if it can be blank, keep for data
    origin VARCHAR(10) NOT NULL,
    destination VARCHAR(10) NOT NULL,
    departure_fix VARCHAR(50) NULL,
    route_string TEXT NULL,
    departure_artcc VARCHAR(10) NULL,
    arrival_artcc VARCHAR(10) NULL,
    traversed_artccs VARCHAR(255) NULL,
    coordination_required VARCHAR(5) NULL, 
    nav_eqp VARCHAR(10) NULL,
    associated_play VARCHAR(255) NULL,

    effective_date_start DATE NULL, 
    effective_date_end DATE NULL,   
    source_file VARCHAR(255) NULL,  

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    -- Removed: UNIQUE KEY uk_cdm_route_code (route_code) 
    -- Add indexes for common query patterns if needed
    INDEX idx_cdm_routes_origin_dest (origin, destination),
    INDEX idx_cdm_routes_route_code (route_code) -- Index route_code for searching, but not unique
);

-- Static Preferred Routes Data (from prefroutes_db.csv)
-- (nfdc_preferred_routes table definition remains the same as before)
DROP TABLE IF EXISTS nfdc_preferred_routes; -- Added for consistency if re-running whole script
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
    effective_date_start DATE NULL,
    effective_date_end DATE NULL,
    source_file VARCHAR(255) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_nfdc_prefroutes_origin_dest (origin, destination),
    INDEX idx_nfdc_prefroutes_effective_dates (effective_date_start, effective_date_end)
);

-- Dynamic Advisory Summaries (from FAA advisory list page)
-- (advisory_summaries table definition remains the same)
DROP TABLE IF EXISTS advisory_details; -- Drop detail first due to FK
DROP TABLE IF EXISTS advisory_summaries; -- Added for consistency
CREATE TABLE IF NOT EXISTS advisory_summaries (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    advisory_date DATE NOT NULL,                 
    summary_unique_key VARCHAR(768) UNIQUE NOT NULL, 
    list_page_raw_text TEXT NOT NULL,            
    issue_time_on_list_page TIME NULL,           
    detail_page_params_json JSON NOT NULL,       
    has_full_detail_saved BOOLEAN NOT NULL DEFAULT FALSE, 
    first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,      
    last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_advisory_summaries_advisory_date (advisory_date),
    INDEX idx_advisory_summaries_has_detail (has_full_detail_saved)
);

-- Dynamic Advisory Details (full text and parsed elements of specific advisories)
-- (advisory_details table definition remains the same)
CREATE TABLE IF NOT EXISTS advisory_details (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    summary_key VARCHAR(768) UNIQUE NOT NULL, 
    source VARCHAR(50) NOT NULL DEFAULT 'FAA_ADVISORY', 
    full_text_content MEDIUMTEXT NOT NULL,     
    parsed_event_time_start_zulu DATETIME NULL,
    parsed_event_time_end_zulu DATETIME NULL,
    parsed_affected_facilities JSON NULL,   
    parsed_remarks TEXT NULL,               
    fetched_and_saved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    FOREIGN KEY (summary_key) REFERENCES advisory_summaries(summary_unique_key) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Table to track versions/effective dates of downloaded static data files
CREATE TABLE IF NOT EXISTS data_source_versions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    source_name VARCHAR(100) UNIQUE NOT NULL, -- e.g., "CDR_CSV", "PREFERRED_ROUTES_CSV"
    source_file_url VARCHAR(512) NULL,
    last_downloaded_filename VARCHAR(255) NULL,
    effective_from DATE NULL,
    effective_until DATE NULL,
    last_checked_on_faa_site_for_eff_date TIMESTAMP NULL,
    last_successfully_downloaded_at TIMESTAMP NULL,
    data_hash VARCHAR(64) NULL, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);