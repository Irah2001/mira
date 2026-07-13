package search

import (
	"mira/internal/notes"
	"strings"
)

func Search(allNotes []notes.Note, query string) []notes.Note {
	var results []notes.Note
	queryLower := strings.ToLower(query)

	for _, n := range allNotes {
		titleLower := strings.ToLower(n.Title)
		contentLower := strings.ToLower(n.Content)

		if strings.Contains(titleLower, queryLower) || strings.Contains(contentLower, queryLower) {
			results = append(results, n)
		}
	}

	return results
}
