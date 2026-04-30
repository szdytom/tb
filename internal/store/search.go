package store

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/szdytom/tb/internal/buffer"
)

const snippetRadius = 60

// SearchResult pairs a matching buffer with a content snippet around the match.
type SearchResult struct {
	Buffer  *buffer.Buffer `json:"buffer"`
	Snippet string         `json:"snippet"`
}

// Search performs full-text search across all active buffers.
// If isRegex is true, query is treated as a Go regular expression.
func (r *Repository) Search(query string, isRegex bool) ([]SearchResult, error) {
	if isRegex {
		return r.searchRegex(query)
	}
	return r.searchLiteral(query)
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
