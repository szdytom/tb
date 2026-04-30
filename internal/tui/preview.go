package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var emptyPreviewStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

type Preview struct {
	lines   []string
	vOffset int
	hOffset int
	width   int
	height  int
}

func NewPreview() *Preview {
	return &Preview{}
}

func (p *Preview) SetContent(content string) {
	p.lines = strings.Split(content, "\n")
	if p.vOffset >= len(p.lines) && len(p.lines) > 0 {
		p.vOffset = len(p.lines) - p.height
		if p.vOffset < 0 {
			p.vOffset = 0
		}
	}
}

func (p *Preview) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *Preview) ScrollUp(n int) {
	p.vOffset -= n
	if p.vOffset < 0 {
		p.vOffset = 0
	}
}

func (p *Preview) ScrollDown(n int) {
	p.vOffset += n
	p.clampOffset()
}

func (p *Preview) PageUp() {
	p.ScrollUp(p.height)
}

func (p *Preview) PageDown() {
	p.ScrollDown(p.height)
}

func (p *Preview) ScrollLeft(n int) {
	p.hOffset -= n
	if p.hOffset < 0 {
		p.hOffset = 0
	}
}

func (p *Preview) ScrollRight(n int) {
	p.hOffset += n
}

func (p *Preview) clampOffset() {
	max := len(p.lines) - p.height
	if max < 0 {
		max = 0
	}
	if p.vOffset > max {
		p.vOffset = max
	}
}

func (p *Preview) View() string {
	if len(p.lines) == 0 || (len(p.lines) == 1 && p.lines[0] == "") {
		return emptyPreviewStyle.Width(p.width).Render("(empty)")
	}

	end := p.vOffset + p.height
	if end > len(p.lines) {
		end = len(p.lines)
	}
	if end <= p.vOffset {
		return ""
	}

	visible := p.lines[p.vOffset:end]
	var b strings.Builder
	for i, line := range visible {
		if i > 0 {
			b.WriteByte('\n')
		}
		absN := p.vOffset + i + 1

		gutter := fmt.Sprintf("%6d | ", absN)
		rem := p.width - ansi.StringWidth(gutter)
		if rem < 0 {
			rem = 0
		}

		content := line
		if p.hOffset < len(line) {
			content = line[p.hOffset:]
		}

		// Truncate to available width
		content = ansi.Truncate(content, rem, "")

		b.WriteString(gutter)
		b.WriteString(content)
	}
	return b.String()
}
