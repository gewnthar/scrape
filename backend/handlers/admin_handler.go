// backend/handlers/admin_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gewnthar/scrape/backend/config" // Adjust to your module path
	"github.com/gewnthar/scrape/backend/services"
)

// Helper to respond with JSON
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON response: %v", err)
		http.Error(w, `{"error":"Failed to marshal JSON response"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// Helper to respond with an error
func respondWithError(w http.ResponseWriter, code int, message string) {
	log.Printf("API Error %d: %s", code, message)
	respondWithJSON(w, code, map[string]string{"error": message})
}

// ForceRefreshStaticRouteDataHandler handles requests to manually refresh static route data.
// Expects POST requests to /api/admin/refresh-routes/{sourceType}
// where {sourceType} is "cdr" or "preferredroutes".
func ForceRefreshStaticRouteDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// Expected path: api/admin/refresh-routes/{sourceType}
	// pathParts: ["api", "admin", "refresh-routes", "{sourceType}"]
	if len(pathParts) < 4 {
		respondWithError(w, http.StatusBadRequest, "Invalid path. Expected /api/admin/refresh-routes/{sourceType}")
		return
	}
	sourceType := strings.ToLower(pathParts[3])

	var err error
	switch sourceType {
	case "cdr":
		err = services.ForceUpdateStaticRouteData("CDR", nil) // nil will trigger live date scraping
	case "preferredroutes":
		err = services.ForceUpdateStaticRouteData("PreferredRoutes", nil)
	case "all":
		err = services.ForceUpdateStaticRouteData("CDR", nil)
		if err == nil { // Only proceed if CDR update was successful
			err = services.ForceUpdateStaticRouteData("PreferredRoutes", nil)
		}
	default:
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid source type '%s'. Use 'cdr', 'preferredroutes', or 'all'.", sourceType))
		return
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to force refresh %s data: %v", sourceType, err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("%s data refresh initiated successfully.", strings.Title(sourceType))})
}

// CheckAndUpdateStaticRouteDataHandler handles requests to check and update static route data if needed.
// Expects POST requests to /api/admin/check-update-routes/{sourceType}
func CheckAndUpdateStaticRouteDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// Expected path: api/admin/check-update-routes/{sourceType}
	if len(pathParts) < 4 {
		respondWithError(w, http.StatusBadRequest, "Invalid path. Expected /api/admin/check-update-routes/{sourceType}")
		return
	}
	sourceType := strings.ToLower(pathParts[3])

	var err error
	var cssSelector string

	switch sourceType {
	case "cdr":
		cssSelector = config.AppConfig.ScraperSelectors.CdrEffectiveDate
		err = services.UpdateStaticRouteDataIfNeeded("CDR", cssSelector)
	case "preferredroutes":
		cssSelector = config.AppConfig.ScraperSelectors.PreferredRoutesEffectiveDate
		err = services.UpdateStaticRouteDataIfNeeded("PreferredRoutes", cssSelector)
	case "all":
		cssSelector = config.AppConfig.ScraperSelectors.CdrEffectiveDate
		err = services.UpdateStaticRouteDataIfNeeded("CDR", cssSelector)
		if err == nil { // Only proceed if CDR was successful (or didn't need update)
			cssSelector = config.AppConfig.ScraperSelectors.PreferredRoutesEffectiveDate
			err = services.UpdateStaticRouteDataIfNeeded("PreferredRoutes", cssSelector)
		}
	default:
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid source type '%s'. Use 'cdr', 'preferredroutes', or 'all'.", sourceType))
		return
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to check/update %s data: %v", sourceType, err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Check/update process for %s data completed.", strings.Title(sourceType))})
}