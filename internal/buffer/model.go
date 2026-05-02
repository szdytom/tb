package buffer

import (
	"strings"
	"time"
)

// TrashStatus represents the lifecycle stage of a buffer.
type TrashStatus int

const (
	TrashStatusActive  TrashStatus = iota // Normal, visible buffer.
	TrashStatusTrashed                    // Moved to trash, awaiting auto-purge.
)

// Metadata holds computed properties about buffer content.
type Metadata struct {
	LineCount int `json:"line_count"`
	ByteCount int `json:"byte_count"`
}

// Buffer is the core data structure representing a text buffer.
type Buffer struct {
	ID          int64       `json:"id"`
	Label       string      `json:"label,omitempty"`
	Content     string      `json:"content"`
	Metadata    Metadata    `json:"metadata"`
	Tags        []string    `json:"tags,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	TrashStatus TrashStatus `json:"trash_status"`
	TrashedAt   *time.Time  `json:"trashed_at,omitempty"`
	ExpiresAt   *time.Time  `json:"expires_at,omitempty"`
}

// NewBuffer creates a new active buffer with the given content and metadata.
func NewBuffer(content, label string, tags []string) *Buffer {
	now := time.Now()

	return &Buffer{
		Content:     content,
		Label:       label,
		Tags:        tags,
		CreatedAt:   now,
		UpdatedAt:   now,
		TrashStatus: TrashStatusActive,
		Metadata:    ComputeMetadata(content),
	}
}

// ComputeMetadata calculates line count and byte count for the given content.
func ComputeMetadata(content string) Metadata {
	lineCount := 0
	if content != "" {
		lineCount = strings.Count(content, "\n")
		if !strings.HasSuffix(content, "\n") {
			lineCount++
		}
	}

	return Metadata{
		LineCount: lineCount,
		ByteCount: len(content),
	}
}

// BufferSummary is a lightweight buffer representation without the full Content.
// Used by the TUI list view for fast loading of many buffers.
type BufferSummary struct {
	ID        int64     `json:"id"`
	Label     string    `json:"label,omitempty"`
	Preview   string    `json:"preview"`
	LineCount int       `json:"line_count"`
	ByteCount int       `json:"byte_count"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FirstLine returns the first line of s (text up to the first newline).
func FirstLine(s string) string {
	if before, _, ok := strings.Cut(s, "\n"); ok {
		return before
	}

	return s
}

// NewBufferSummary creates a BufferSummary from a Buffer.
func NewBufferSummary(b *Buffer) BufferSummary {
	preview := FirstLine(b.Content)
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}

	return BufferSummary{
		ID:        b.ID,
		Label:     b.Label,
		Preview:   preview,
		LineCount: b.Metadata.LineCount,
		ByteCount: b.Metadata.ByteCount,
		Tags:      b.Tags,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}
