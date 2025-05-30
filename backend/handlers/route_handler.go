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
)

// Re-defining these helpers here for now. Consider moving to a common utils package
// if you haven't already defined them in another handler file you're actively using.
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
// Expects POST to /api/routes/find
// with JSON body: {"origin": "JFK", "destination": "MIA", "date": "YYYY-MM-DD"}
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

	// Basic validation
	if req.Origin == "" {
		respondWithError_Route(w, http.StatusBadRequest, "Missing 'origin' in request body")
		return
	}
	if req.Destination == "" {
		respondWithError_Route(w, http.StatusBadRequest, "Missing 'destination' in request body")
		return
	}
	if req.Date == "" {
		respondWithError_Route(w, http.StatusBadRequest, "Missing 'date' in request body")
		return
	}

	queryDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		respondWithError_Route(w, http.StatusBadRequest, "Invalid date format for 'date'. Use YYYY-MM-DD. Error: "+err.Error())
		return
	}

	log.Printf("Handler: Received find routes request for %s-%s on %s\n", req.Origin, req.Destination, queryDate.Format("2006-01-02"))

	serviceInput := services.FindBestRoutesInput{
		Origin:      req.Origin,
		Destination: req.Destination,
		QueryDate:   queryDate,
	}

	recommendedRoutes, err := services.FindBestRoutes(serviceInput)
	if err != nil {
		respondWithError_Route(w, http.StatusInternalServerError, fmt.Sprintf("Failed to find routes: %v", err))
		return
	}

	if recommendedRoutes == nil { // Ensure we always return an array for JSON, even if empty
		recommendedRoutes = []models.RecommendedRoute{}
	}

	respondWithJSON_Route(w, http.StatusOK, recommendedRoutes)
}