// backend/services/route_service.go
package services

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/gewnthar/scrape/backend/database"
	"github.com/gewnthar/scrape/backend/models"
	// "github.com/gewnthar/scrape/backend/config" // If needed for specific service configs
)

// RouteFinderService orchestrates finding the best routes.
// For now, using package-level functions. A struct could be used for dependency injection.

// FindBestRoutesInput holds parameters for finding routes.
type FindBestRoutesInput struct {
	Origin      string
	Destination string
	QueryDate   time.Time
}

// FindBestRoutes analyzes advisories, CDRs, and Preferred Routes to recommend flight paths.
func FindBestRoutes(input FindBestRoutesInput) ([]models.RecommendedRoute, error) {
	log.Printf("Service: Finding best routes for %s-%s on %s\n", input.Origin, input.Destination, input.QueryDate.Format("2006-01-02"))
	var recommendations []models.RecommendedRoute

	// --- Step 1: Fetch Relevant Active Advisories ---
	// This function needs to be fully implemented in advisory_service.go
	// For now, assume it might return a list of *all* summaries for the day,
	// and we'd need to filter them here or have the service do more specific filtering.
	// Let's assume GetAndDisplayAdvisorySummaries gives us relevant summaries (e.g., for the date).
	// We will need to parse their full details to find routing instructions.
	
	// Fetch summaries for the query date (includes potential live scrape)
	advisorySummaries, err := GetAndDisplayAdvisorySummaries(input.QueryDate, "") // Empty keyword gets all for date
	if err != nil {
		log.Printf("WARN RouteService: Could not fetch advisory summaries for %s: %v. Proceeding without live advisories.", input.QueryDate.Format("2006-01-02"), err)
		// Continue, but recommendations will be based only on static routes if this fails.
	}
	
	var activeRouteAdvisories []models.AdvisoryDetail
	directiveFileCDRs := false
	// TODO: Placeholder for Reroute Advisories from RAT Reader
	// ratReroutes, _ := scraper.FetchRatReroutesForOD(input.Origin, input.Destination, input.QueryDate)

	// Process summaries to get details and look for routing instructions
	for _, summary := range advisorySummaries {
		// For high-priority advisories (RQD, specific keywords), fetch detail
		// This is a simplified check; real logic would be more nuanced.
		if strings.Contains(strings.ToUpper(summary.ListPageRawText), "_RQD") || 
		   strings.Contains(strings.ToUpper(summary.ListPageRawText), "ROUTE") ||
		   strings.Contains(strings.ToUpper(summary.ListPageRawText), "CDR") ||
		   strings.Contains(strings.ToUpper(summary.ListPageRawText), "SWAP") {
			
			// Get detail (from DB or scrape if needed)
			// Note: GetOrFetchAdvisoryDetail in advisory_service.go gets detail but doesn't auto-save it.
			// For route finding, we need the content.
			detail, detailErr := GetOrFetchAdvisoryDetail(summary.SummaryUniqueKey, summary.DetailPageParams)
			if detailErr != nil {
				log.Printf("WARN RouteService: Failed to get detail for summary %s: %v", summary.SummaryUniqueKey, detailErr)
				continue
			}
			if detail != nil {
				// Basic check for "File CDRs" directive
				if strings.Contains(strings.ToUpper(detail.FullTextContent), "FILE CDRS") || strings.Contains(strings.ToUpper(detail.FullTextContent), "CDR PLAYBOOK") {
					directiveFileCDRs = true
					log.Printf("INFO RouteService: Advisory %s indicates 'File CDRs' or 'CDR Playbook'.\n", detail.SummaryKey)
				}
				// Basic check for RQD routes (more specific parsing needed here)
				if strings.Contains(strings.ToUpper(summary.ListPageRawText), "_RQD") { // Check summary again too
					// TODO: Extract actual route string from RQD advisory text
					// For now, just flag it and add the advisory text.
					recommendations = append(recommendations, models.RecommendedRoute{
						Origin:        input.Origin, // Or from advisory if more specific
						Destination:   input.Destination,
						RouteString:   "PER RQD ADVISORY - SEE TEXT", // Placeholder
						Source:        "RQD Advisory",
						Justification: fmt.Sprintf("Advisory %s: %s", detail.SummaryKey, summary.ListPageRawText),
						FullAdvisory:  detail,
						Priority:      models.PriorityRqdAdvisory,
						Restrictions:  "Refer to full advisory text for details and restrictions.",
					})
				}
				activeRouteAdvisories = append(activeRouteAdvisories, *detail)
			}
		}
	}
	
	// If an RQD advisory directly specified a route, it might be the only one to show.
	// For this version, we'll collect all and sort.
	// TODO: Add logic for RAT Reroutes - these would have high priority if active.

	// --- Step 2: Fetch Static Routes (Preferred & CDRs) from our DB ---
	preferredRoutes, err := database.GetPreferredRoutesForOD(input.Origin, input.Destination, input.QueryDate)
	if err != nil {
		log.Printf("WARN RouteService: Failed to get preferred routes for %s-%s: %v\n", input.Origin, input.Destination, err)
	} else {
		for _, pr := range preferredRoutes {
			rec := models.RecommendedRoute{
				Origin:         pr.Origin,
				Destination:    pr.Destination,
				RouteString:    pr.RouteString,
				Source:         "Preferred Route",
				Justification:  fmt.Sprintf("Type: %s, Seq: %s", pr.Type, pr.Sequence),
				Preferred:      &pr, // Keep a pointer to the original struct
				Priority:       models.PriorityPreferredRoute,
				Restrictions:   fmt.Sprintf("Aircraft: %s; Alt: %s; Hours: %s/%s/%s", pr.Aircraft, pr.Altitude, pr.Hours1, pr.Hours2, pr.Hours3),
				EffectiveStart: pr.EffectiveDateStart,
				EffectiveEnd:   pr.EffectiveDateEnd,
			}
			recommendations = append(recommendations, rec)
		}
	}

	cdrRoutes, err := database.GetCdrRoutesForOD(input.Origin, input.Destination, input.QueryDate)
	if err != nil {
		log.Printf("WARN RouteService: Failed to get CDRs for %s-%s: %v\n", input.Origin, input.Destination, err)
	} else {
		for _, cdr := range cdrRoutes {
			priority := models.PriorityCdrCoord
			sourceSuffix := " (Coord Req)"
			if strings.ToUpper(cdr.CoordinationRequired) == "N" {
				priority = models.PriorityCdrNoCoord
				sourceSuffix = " (No Coord)"
			}
			
			// If advisory says "File CDRs", elevate CDR priority
			if directiveFileCDRs {
				priority = models.PriorityFileCDRsDirective 
				// Could further refine priority based on coord status even with directive
				if strings.ToUpper(cdr.CoordinationRequired) == "N" {
					priority -= 2 // Slightly better
				} else {
					priority -=1 
				}
			}


			rec := models.RecommendedRoute{
				Origin:         cdr.Origin,
				Destination:    cdr.Destination,
				RouteString:    cdr.RouteString,
				Source:         "CDR" + sourceSuffix,
				Justification:  fmt.Sprintf("Code: %s, Play: %s", cdr.RouteCode, cdr.AssociatedPlay),
				Cdr:            &cdr, // Keep a pointer
				Priority:       priority,
				Restrictions:   fmt.Sprintf("NavEqp: %s", cdr.NavEqp),
				EffectiveStart: cdr.EffectiveDateStart,
				EffectiveEnd:   cdr.EffectiveDateEnd,
			}
			recommendations = append(recommendations, rec)
		}
	}

	// --- Step 3: Apply Prioritization and Sort ---
	// (More complex filtering based on advisory content vs. static routes would go here)
	// E.g., if an RQD advisory invalidates certain preferred routes or CDRs.

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority < recommendations[j].Priority
	})

	log.Printf("Service: Found %d potential route recommendations for %s-%s, sorted by priority.\n", len(recommendations), input.Origin, input.Destination)
	return recommendations, nil
}