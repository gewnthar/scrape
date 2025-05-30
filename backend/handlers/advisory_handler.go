// backend/handlers/advisory_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url" // For parsing query parameters for detail
	"strings"
	"time"

	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
	"github.com/gewnthar/scrape/backend/services"
)

// Re-defining these helpers here for now. Consider moving to a common utils package.
func respondWithJSON_Advisory(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("AdvisoryHandler ERROR: Marshalling JSON response: %v", err)
		http.Error(w, `{"error":"Failed to marshal JSON response"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError_Advisory(w http.ResponseWriter, code int, message string) {
	log.Printf("AdvisoryHandler API Error %d: %s", code, message)
	respondWithJSON_Advisory(w, code, map[string]string{"error": message})
}

// --- Scenario A Handler ---

// ExploratoryAdvisorySearchHandler handles requests for Scenario A:
// Fetches advisories for a specific operational window (day 00Z to day+1 08Z)
// and auto-saves both summaries and their full details.
// Expects POST to /api/advisories/exploratory-search with JSON body: {"date": "YYYY-MM-DD"}
func ExploratoryAdvisorySearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError_Advisory(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	var requestBody struct {
		Date string `json:"date"` // Expected format "YYYY-MM-DD"
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		respondWithError_Advisory(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	if requestBody.Date == "" {
		respondWithError_Advisory(w, http.StatusBadRequest, "Missing 'date' in request body")
		return
	}

	targetDate, err := time.Parse("2006-01-02", requestBody.Date)
	if err != nil {
		respondWithError_Advisory(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD. Error: "+err.Error())
		return
	}

	log.Printf("Handler: Received exploratory search request for date: %s\n", targetDate.Format("2006-01-02"))

	go func() { // Run processing in a goroutine so the HTTP request can return quickly
		err := services.ProcessExploratoryAdvisorySearch(targetDate)
		if err != nil {
			// Log the error. Since this is async, we can't easily return HTTP error to original client.
			// Consider a notification system or status endpoint for long-running tasks.
			log.Printf("ERROR Handler: Exploratory search for %s failed: %v\n", targetDate.Format("2006-01-02"), err)
		} else {
			log.Printf("Handler: Exploratory search for %s completed successfully (background task).\n", targetDate.Format("2006-01-02"))
		}
	}()

	respondWithJSON_Advisory(w, http.StatusAccepted, map[string]string{
		"message": fmt.Sprintf("Exploratory advisory search for %s initiated in background.", targetDate.Format("2006-01-02")),
	})
}

// --- Scenario B Handlers ---

// GetAdvisorySummariesHandler fetches advisory summaries for a date, optionally filtered by keyword.
// Expects GET to /api/advisories?date=YYYY-MM-DD[&keyword=...]
func GetAdvisorySummariesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError_Advisory(w, http.StatusMethodNotAllowed, "Only GET method is allowed")
		return
	}

	dateStr := r.URL.Query().Get("date")
	keyword := r.URL.Query().Get("keyword")

	if dateStr == "" {
		respondWithError_Advisory(w, http.StatusBadRequest, "Missing 'date' query parameter")
		return
	}

	queryDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondWithError_Advisory(w, http.StatusBadRequest, "Invalid date format for 'date' query parameter. Use YYYY-MM-DD.")
		return
	}

	log.Printf("Handler: Received get summaries request for date: %s, keyword: '%s'\n", queryDate.Format("2006-01-02"), keyword)

	summaries, err := services.GetAndDisplayAdvisorySummaries(queryDate, keyword)
	if err != nil {
		respondWithError_Advisory(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get advisory summaries: %v", err))
		return
	}

	if summaries == nil { // Ensure we always return an array, even if empty
		summaries = []models.AdvisorySummary{}
	}
	respondWithJSON_Advisory(w, http.StatusOK, summaries)
}

// GetAdvisoryDetailHandler fetches a specific advisory detail.
// It tries the DB first, then scrapes from FAA if not found (returns for viewing, does not auto-save).
// Expects GET to /api/advisories/detail?summary_key=UNIQUE_KEY
// OR GET to /api/advisories/detail?advn=X&adv_date=Y&facId=Z (passing through detail params)
func GetAdvisoryDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError_Advisory(w, http.StatusMethodNotAllowed, "Only GET method is allowed")
		return
	}

	queryParams := r.URL.Query()
	summaryKey := queryParams.Get("summary_key")
	
	var detailParams map[string]string
	if summaryKey == "" { 
		// If no summary_key, try to build detailParams from query (less ideal but a fallback)
		// The service layer will need to be robust if it gets a nil detailParams if summaryKey is also missing
		// and summaryKey is preferred to ensure we link to a known summary.
		// For now, let's assume summaryKey is the primary way, or client sends detail_page_params as JSON in POST if complex.
		// Or, if we must use GET, client could pass a base64 encoded JSON string of params.
		// Let's simplify: client gets summaries, then uses summaryKey.
		// If scraping ad-hoc without a summary, a different endpoint might be better.
		// For now, if summaryKey is empty, we'll assume it's an error for THIS handler.
		// The service `GetOrFetchAdvisoryDetail` expects detailParams if summaryKey leads to DB miss.
		// This handler needs to provide those params to the service.
		// A better approach for GetOrFetchAdvisoryDetail might be to take summaryKey,
		// and the service itself looks up the params from the advisory_summaries table.
		
		// For this version, we require summaryKey to fetch detail params from summary table first
		// Or, if this handler is meant to scrape "blindly" from params, it should be a POST with JSON body.
		// Let's assume for now client provides summaryKey.
		// The service layer `GetOrFetchAdvisoryDetail` will need to fetch the summary by key to get its params if needed.
		// This handler will just pass the summaryKey.
		
		// Alternative: Pass all detail params in query for a "direct scrape" if key unknown
		// This is complex for GET due to URL encoding of many params.
		// For now, we will rely on summary_key.
		// The service will look up params if scraping is needed.
		// This means the service layer will need `GetAdvisorySummaryByKey`
		// Let's stick to `summary_key` only for this GET handler for simplicity.
		// The service `GetOrFetchAdvisoryDetail` should be robust enough.
		if summaryKey == "" {
			respondWithError_Advisory(w, http.StatusBadRequest, "Missing 'summary_key' query parameter")
			return
		}
	}

	log.Printf("Handler: Received get detail request for summary_key: %s\n", summaryKey)
	
	// The service function GetOrFetchAdvisoryDetail will look up detailParams from summaryKey if needed.
	detail, err := services.GetOrFetchAdvisoryDetail(summaryKey, nil) // Pass nil for detailParams; service should fetch if needed from summaryKey
	if err != nil {
		respondWithError_Advisory(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get advisory detail for summary_key %s: %v", summaryKey, err))
		return
	}

	if detail == nil {
		respondWithError_Advisory(w, http.StatusNotFound, fmt.Sprintf("Advisory detail not found for summary_key %s", summaryKey))
		return
	}

	respondWithJSON_Advisory(w, http.StatusOK, detail)
}

// ConfirmSaveAdvisoryDetailHandler saves a provided advisory detail to the database.
// Expects POST to /api/advisories/detail/save with JSON body of models.AdvisoryDetail
func ConfirmSaveAdvisoryDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError_Advisory(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	var detail models.AdvisoryDetail
	if err := json.NewDecoder(r.Body).Decode(&detail); err != nil {
		respondWithError_Advisory(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	if detail.SummaryKey == "" {
		respondWithError_Advisory(w, http.StatusBadRequest, "Missing 'SummaryKey' in advisory detail")
		return
	}
	if detail.FullTextContent == "" {
		respondWithError_Advisory(w, http.StatusBadRequest, "Missing 'FullTextContent' in advisory detail")
		return
	}

	log.Printf("Handler: Received confirm save detail request for summary_key: %s\n", detail.SummaryKey)

	err := services.ConfirmSaveAdvisoryDetail(detail)
	if err != nil {
		respondWithError_Advisory(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save advisory detail for %s: %v", detail.SummaryKey, err))
		return
	}

	respondWithJSON_Advisory(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Advisory detail for %s saved successfully.", detail.SummaryKey)})
}