package store

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/szdytom/tb/internal/buffer"
)

// SearchMode specifies the search algorithm.
type SearchMode string

const (
	SearchModeFuzzy   SearchMode = "fuzzy"
	SearchModeLiteral SearchMode = "literal"
	SearchModeRegex   SearchMode = "regex"
)

const snippetRadius = 60

// SearchResult pairs a matching buffer with a content snippet around the match.
type SearchResult struct {
	Buffer  *buffer.Buffer `json:"buffer"`
	Snippet string         `json:"snippet"`
}

// Search performs full-text search across all active buffers.
// mode controls the search algorithm: "fuzzy", "literal", or "regex".
func (r *Repository) Search(query string, mode SearchMode) ([]SearchResult, error) {
	switch mode {
	case SearchModeFuzzy:
		return r.searchFuzzy(query)
	case SearchModeRegex:
		return r.searchRegex(query)
	default:
		return r.searchLiteral(query)
	}
}

func (r *Repository) searchLiteral(query string) ([]SearchResult, error) {
	rows, err := r.db.Query(`
		SELECT id, label, content, line_count, byte_count, tags, created_at, updated_at, trash_status, trashed_at, expires_at
		FROM buffers WHERE trash_status = ? AND content LIKE '%' || ? || '%'
		ORDER BY updated_at DESC`, buffer.TrashStatusActive, query)
	if err != nil {
		return nil, fmt.Errorf("literal search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		buf, err := scanBuffer(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, SearchResult{
			Buffer:  buf,
			Snippet: extractSnippet(buf.Content, query, nil),
		})
	}
	return results, rows.Err()
}

func (r *Repository) searchRegex(pattern string) ([]SearchResult, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
	}

	// Fetch all active buffers and filter in Go.
	rows, err := r.db.Query(`
		SELECT id, label, content, line_count, byte_count, tags, created_at, updated_at, trash_status, trashed_at, expires_at
		FROM buffers WHERE trash_status = ? ORDER BY updated_at DESC`, buffer.TrashStatusActive)
	if err != nil {
		return nil, fmt.Errorf("regex search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		buf, err := scanBuffer(rows)
		if err != nil {
			return nil, err
		}
		if re.MatchString(buf.Content) {
			results = append(results, SearchResult{
				Buffer:  buf,
				Snippet: extractSnippet(buf.Content, pattern, re),
			})
		}
	}
	return results, rows.Err()
}

func (r *Repository) searchFuzzy(query string) ([]SearchResult, error) {
	rows, err := r.db.Query(`
		SELECT id, label, content, line_count, byte_count, tags, created_at, updated_at, trash_status, trashed_at, expires_at
		FROM buffers WHERE trash_status = ? ORDER BY updated_at DESC`, buffer.TrashStatusActive)
	if err != nil {
		return nil, fmt.Errorf("fuzzy search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		buf, err := scanBuffer(rows)
		if err != nil {
			return nil, err
		}
		if fuzzy.FindFold(query, []string{buf.Content}) != nil {
			results = append(results, SearchResult{
				Buffer:  buf,
				Snippet: extractFuzzySnippet(buf.Content, query),
			})
		}
	}
	return results, rows.Err()
}

// extractFuzzySnippet returns a portion of content near the first character
// of the query, since fuzzy matches are non-contiguous.
func extractFuzzySnippet(content, query string) string {
	if content == "" || query == "" {
		return truncateHead(content)
	}
	lowerContent := strings.ToLower(content)
	idx := strings.Index(lowerContent, strings.ToLower(string(query[0])))
	if idx == -1 {
		return truncateHead(content)
	}
	start := idx - snippetRadius
	if start < 0 {
		start = 0
	}
	end := idx + snippetRadius
	if end > len(content) {
		end = len(content)
	}
	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}
	return snippet
}

// extractSnippet returns a portion of content surrounding the first match of query.
// If re is non-nil, it is used for locating the match (regex); otherwise literal matching is used.
func extractSnippet(content, query string, re *regexp.Regexp) string {
	if content == "" {
		return ""
	}

	var idx int
	if re != nil {
		loc := re.FindStringIndex(content)
		if loc == nil {
			return truncateHead(content)
		}
		idx = loc[0]
	} else {
		lowerContent := strings.ToLower(content)
		lowerQuery := strings.ToLower(query)
		idx = strings.Index(lowerContent, lowerQuery)
		if idx == -1 {
			return truncateHead(content)
		}
	}

	start := idx - snippetRadius
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + snippetRadius
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}
	return snippet
}

func truncateHead(content string) string {
	if len(content) <= snippetRadius*2 {
		return content
	}
	return content[:snippetRadius*2] + "..."
}
