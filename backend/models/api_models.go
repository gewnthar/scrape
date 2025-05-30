// backend/models/api_models.go
package models

// FindRoutesRequest is the expected JSON body for the /api/routes/find endpoint.
type FindRoutesRequest struct {
	Origin      string `json:"origin"`       // e.g., "JFK"
	Destination string `json:"destination"`  // e.g., "MIA"
	Date        string `json:"date"`         // Expected format "YYYY-MM-DD"
}

// You can add other API-specific request/response models here later.
// For example, a specific response struct for FindRoutes if needed,
// though returning []RecommendedRoute directly is often fine.