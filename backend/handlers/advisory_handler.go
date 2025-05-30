// backend/handlers/advisory_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// ExploratoryAdvisorySearchHandler (code remains the same as before)
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

	go func() { 
		err := services.ProcessExploratoryAdvisorySearch(targetDate)
		if err != nil {
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

// GetAdvisorySummariesHandler (code remains the same as before)
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

	if summaries == nil { 
		summaries = []models.AdvisorySummary{}
	}
	respondWithJSON_Advisory(w, http.StatusOK, summaries)
}

// GetAdvisoryDetailHandler fetches a specific advisory detail.
// It tries the DB first, then scrapes from FAA if not found (returns for viewing, does not auto-save).
// Expects GET to /api/advisories/detail?summary_key=UNIQUE_KEY
func GetAdvisoryDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError_Advisory(w, http.StatusMethodNotAllowed, "Only GET method is allowed")
		return
	}

	queryParams := r.URL.Query()
	summaryKey := queryParams.Get("summary_key")
	
	// Removed the unused 'detailParams map[string]string' declaration
	
	if summaryKey == "" { 
		respondWithError_Advisory(w, http.StatusBadRequest, "Missing 'summary_key' query parameter")
		return
	}

	log.Printf("Handler: Received get detail request for summary_key: %s\n", summaryKey)
	
	// Pass nil for detailParams; service (GetOrFetchAdvisoryDetail) should fetch 
	// the necessary parameters from the advisory_summaries table using summaryKey if a live scrape is needed.
	detail, err := services.GetOrFetchAdvisoryDetail(summaryKey, nil) 
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

// ConfirmSaveAdvisoryDetailHandler (code remains the same as before)
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