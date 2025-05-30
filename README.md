# Scrape Advisory - FAA Decision Support Tool (DST)

## Overview

Scrape Advisory is a Decision Support Tool designed to fetch, process, store, and analyze aviation data from various public FAA (Federal Aviation Administration) sources. The primary goal is to make traditionally non-machine-friendly advisory and route information more accessible, queryable, and useful for operational decision-making, such as determining optimal flight routes based on current conditions and regulations.

This tool addresses the challenge of consolidating dynamic advisories with more static route databases (like CDRs and Preferred Routes) to provide a clearer operational picture.

## Key Features

* **Data Aggregation:**
    * Scrapes general FAA flight advisories (including Airport Status, Airspace Flow Programs).
    * Downloads and processes Coded Departure Routes (CDRs) from FAA CSV data.
    * Downloads and processes NFDC Preferred Routes from FAA CSV data.
    * (Planned) Scrapes FAA Reroute Advisories (RAT Reader).
* **Persistent Storage:** Stores all collected data in a MariaDB database for historical access and complex querying.
* **Intelligent Data Updates:**
    * Semi-automated updates for CDR and Preferred Route CSVs by checking their "Effective Until" dates on FAA websites.
    * Incorporates knowledge of the 56-day publication cycle for FAA data.
    * Manual refresh capability for all static data sources via API.
* **Core Decision Support Logic:**
    * Analyzes current advisories in conjunction with stored CDRs and Preferred Routes for a given Origin/Destination pair.
    * Prioritizes routes based on advisory mandates (RQD, "File CDRs" directives, FCA conditions) and route characteristics (e.g., CDRs requiring coordination).
* **API Access:** Provides a Go-based backend API to interact with the processed data and trigger actions.
* **User Interface:** (Planned) A web-based frontend built with HTML, CSS, and Vanilla JavaScript for user interaction.

## Tech Stack

* **Backend:** Go
    * HTTP Server: `net/http` (standard library)
    * Database Driver: `go-sql-driver/mysql` (for MariaDB)
    * CSV Parsing: `github.com/jszwec/csvutil`
    * HTML Parsing: `github.com/PuerkitoBio/goquery`
    * Configuration: YAML
* **Database:** MariaDB
* **Frontend:** HTML, CSS, Vanilla JavaScript
* **Deployment Environment (User's):** AlmaLinux VPS, Apache (as reverse proxy), CSF Firewall, Cloudflare.

## Project Structure Overview

The project is organized into several main directories:
* `backend/`: Contains all the Go source code for the API server, data scraping, database interaction, and business logic. This is further subdivided into packages like `config`, `database`, `handlers`, `models`, `scraper`, and `services`.
* `frontend/`: Will hold all static assets (HTML, CSS, JavaScript) for the user interface.
* `database_schemas/`: Contains the `schema.sql` file with DDL statements to create the necessary database tables.

## Setup and Installation

### Prerequisites

* Go (version 1.18+ recommended)
* MariaDB (or MySQL compatible) server
* Git

### 1. Clone Repository

```bash
git clone [https://github.com/gewnthar/scrape.git](https://github.com/gewnthar/scrape.git) 
cd scrape