// backend/handlers/route_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gewnthar/scrape/backend/models" // Adjust to your module path
	"github.com/gewnthar/scrape/backend/services"
	"github.com/gewnthar/scrape/backend/utils" // Assuming you created utils/airports.go
)

// respondWithJSON_Route and respondWithError_Route helpers (as before) ...
func respondWithJSON_Route(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("RouteHandler ERROR: Marshalling JSON response: %v", err)
		http.Error(w, `{"error":"Failed to marshal JSON response"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError_Route(w http.ResponseWriter, code int, message string) {
	log.Printf("RouteHandler API Error %d: %s", code, message)
	respondWithJSON_Route(w, code, map[string]string{"error": message})
}


// FindRoutesHandler handles requests to find and prioritize routes.
func FindRoutesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError_Route(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	var req models.FindRoutesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError_Route(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	if req.Origin == "" {
		respondWithError_Route(w, http.StatusBadRequest, "Missing 'origin' in request body")
		return
	}
	if req.Destination == "" {
		respondWithError_Route(w, http.StatusBadRequest, "Missing 'destination' in request body")
		return
	}

	var queryDate time.Time
	var err error
	if req.Date == "" {
		queryDate = time.Now().UTC().Truncate(24 * time.Hour)
		log.Printf("Handler: No date provided for FindRoutes, defaulting to today (UTC): %s\n", queryDate.Format("2006-01-02"))
	} else {
		queryDate, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			respondWithError_Route(w, http.StatusBadRequest, "Invalid date format for 'date'. Use YYYY-MM-DD. Error: "+err.Error())
			return
		}
	}

	// Normalize airport codes from user input
	normalizedOrigin := utils.NormalizeAirportCode(req.Origin)
	normalizedDestination := utils.NormalizeAirportCode(req.Destination)

	log.Printf("Handler: Received find routes request for %s-%s (normalized: %s-%s) on %s\n", 
		req.Origin, req.Destination, 
		normalizedOrigin, normalizedDestination, 
		queryDate.Format("2006-01-02"))

	serviceInput := services.FindBestRoutesInput{
		Origin:      normalizedOrigin,    // Use normalized
		Destination: normalizedDestination, // Use normalized
		QueryDate:   queryDate,
	}

	recommendedRoutes, err := services.FindBestRoutes(serviceInput)
	if err != nil {
		respondWithError_Route(w, http.StatusInternalServerError, fmt.Sprintf("Failed to find routes: %v", err))
		return
	}

	if recommendedRoutes == nil { 
		recommendedRoutes = []models.RecommendedRoute{}
	}

	respondWithJSON_Route(w, http.StatusOK, recommendedRoutes)
}