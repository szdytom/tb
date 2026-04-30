package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var helpBox = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	Padding(1, 2).
	Width(40)

func HelpView(width, height int) string {
	lines := []string{
		" j / Up        Move selection up",
		" k / Down      Move selection down",
		" PgUp / PgDn   Page up / down",
		" g / G         Top / bottom",
		" Enter         Open in editor",
		" n             New buffer",
		" d             Delete buffer",
		" /             Search",
		" :q / Ctrl+C   Quit",
		" ?             Toggle help",
	}
	content := strings.Join(lines, "\n")
	box := helpBox.Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
