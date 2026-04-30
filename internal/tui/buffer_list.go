package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/szdytom/tb/internal/buffer"
)

var (
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	labelStyle    = lipgloss.NewStyle().Faint(true)
	headerStyle   = lipgloss.NewStyle().Underline(true).Bold(true)
	emptyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
)

// RenderList renders the buffer list with a header row and buffer entries.
// The first line is the header; remaining lines show up to (height-1) entries.
// Caller pads to column width via padWidth.
func RenderList(summaries []buffer.BufferSummary, cursor, offset, height, width int) string {
	if len(summaries) == 0 {
		return emptyStyle.Width(width).Render("No buffers. Press 'n' to create one.")
	}

	var lines []string
	lines = append(lines, renderHeader(width))

	maxEntries := height - 1
	if maxEntries < 0 {
		maxEntries = 0
	}
	end := offset + maxEntries
	if end > len(summaries) {
		end = len(summaries)
	}
	for i := offset; i < end; i++ {
		lines = append(lines, renderEntry(summaries[i], i == cursor))
	}
	return strings.Join(lines, "\n")
}

func renderHeader(width int) string {
	h := fmt.Sprintf("%-6s  %-6s  %s", "ID", "Time", "Preview")
	return headerStyle.Width(width).Render(h)
}

func renderEntry(s buffer.BufferSummary, selected bool) string {
	preview := s.Preview
	if preview == "" {
		preview = "(empty)"
	}
	// Ensure preview is a single line (SUBSTR in SQL may include newlines)
	if idx := strings.IndexByte(preview, '\n'); idx >= 0 {
		preview = preview[:idx]
	}

	var line strings.Builder
	line.WriteString(fmt.Sprintf("#%-5d", s.ID))
	line.WriteString("  ")
	line.WriteString(fmt.Sprintf("%-6s", relativeTime(s.UpdatedAt)))
	line.WriteString("  ")
	line.WriteString(preview)

	if s.Label != "" {
		line.WriteString("  ")
		line.WriteString(labelStyle.Render(s.Label))
	}

	rendered := line.String()
	if selected {
		rendered = selectedStyle.Render(rendered)
	}
	return rendered
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours())/24)
	default:
		return t.Format("01-02")
	}
}
