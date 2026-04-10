// search.go — fuzzy session search for telescope / attach picker.
package session

import "strings"

// Search filters session summaries by fuzzy matching on name, workspace, and model.
// Returns matches sorted by recency (same order as input).
func Search(summaries []SessionSummary, query string) []SessionSummary {
	if query == "" {
		return summaries
	}
	query = strings.ToLower(query)

	var matches []SessionSummary
	for i := range summaries {
		haystack := strings.ToLower(strings.Join([]string{
			summaries[i].Name, summaries[i].ID, summaries[i].Model, summaries[i].Driver, summaries[i].WorkDir,
		}, " "))
		if strings.Contains(haystack, query) {
			matches = append(matches, summaries[i])
		}
	}
	return matches
}
