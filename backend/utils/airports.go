// backend/utils/airports.go
package utils

import "strings"

// NormalizeAirportCode converts 4-letter US ICAO codes (e.g., "KJFK") to 3-letter codes ("JFK").
// Other codes are returned as is. Converts to uppercase.
func NormalizeAirportCode(code string) string {
	upperCode := strings.ToUpper(strings.TrimSpace(code))
	if len(upperCode) == 4 && strings.HasPrefix(upperCode, "K") {
		return upperCode[1:]
	}
	return upperCode
}