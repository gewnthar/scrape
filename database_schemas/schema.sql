-- database_schemas/schema.sql

-- Add this if the file is new, or append if other tables are already defined.
-- Make sure your database is selected, e.g., USE faa_dst_db;

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

    -- Consider adding effective dates if the whole CSV file has a common period
    effective_date_start DATE NULL,
    effective_date_end DATE NULL,
    source_file VARCHAR(255) NULL, -- Name of the CSV file it came from, maybe with a date

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    -- A unique key to prevent duplicate route entries if route_code should be unique
    UNIQUE KEY uk_cdm_route_code (route_code) 
);

CREATE TABLE IF NOT EXISTS nfdc_preferred_routes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    
    origin VARCHAR(10) NOT NULL,
    route_string TEXT NULL,
    destination VARCHAR(10) NOT NULL,
    hours1 VARCHAR(255) NULL,
    hours2 VARCHAR(255) NULL,
    hours3 VARCHAR(255) NULL,
    route_type VARCHAR(20) NULL, -- e.g., H, L, TEC
    area VARCHAR(255) NULL,
    altitude VARCHAR(255) NULL,
    aircraft TEXT NULL, -- Can be lengthy
    direction VARCHAR(100) NULL,
    sequence VARCHAR(50) NULL, -- Or INT if it's always a number
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