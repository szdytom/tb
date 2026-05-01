package tui

import (
	"fmt"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"github.com/szdytom/tb/internal/buffer"
)

// DrawList renders the buffer list into a vaxis window.
// The window should have contentH rows. Row 0 is the header.
func DrawList(win vaxis.Window, summaries []buffer.BufferSummary, cursor, listOff, contentH int) {
	w, _ := win.Size()

	if len(summaries) == 0 {
		win.Println(0, vaxis.Segment{
			Text:  "No buffers. Press 'n' to create one.",
			Style: emptyStyle,
		})
		return
	}

	// Header row
	win.PrintTruncate(0, vaxis.Segment{
		Text:  fmt.Sprintf("%-6s  %-6s  %s", "ID", "Time", "Preview"),
		Style: headerStyle,
	})

	maxEntries := contentH - 1
	if maxEntries < 0 {
		maxEntries = 0
	}

	// Truncate preview to available width minus the header prefix width
	// Header prefix: "#NNNNN  MMMMM  " — use 16 chars approximation
	availWidth := w - 16
	if availWidth < 10 {
		availWidth = 10
	}

	end := listOff + maxEntries
	if end > len(summaries) {
		end = len(summaries)
	}

	row := 1
	for i := listOff; i < end; i++ {
		s := summaries[i]
		preview := s.Preview
		if preview == "" {
			preview = "(empty)"
		}
		if len(preview) > availWidth {
			preview = preview[:availWidth]
		}

		var line string
		if s.Label != "" {
			line = fmt.Sprintf("#%-5d  %-6s  %s  %s", s.ID, relativeTime(s.UpdatedAt), preview, s.Label)
		} else {
			line = fmt.Sprintf("#%-5d  %-6s  %s", s.ID, relativeTime(s.UpdatedAt), preview)
		}

		style := vaxis.Style{}
		if i == cursor {
			style.Attribute = vaxis.AttrReverse
		}

		win.PrintTruncate(row, vaxis.Segment{Text: line, Style: style})
		row++
	}
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
