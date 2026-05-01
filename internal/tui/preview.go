package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/term"
)

type TextPreview struct {
	lines   []string
	vOffset int
	hOffset int
	width   int
	height  int
}

func NewTextPreview() *TextPreview {
	return &TextPreview{}
}

func (p *TextPreview) SetContent(content string) {
	p.lines = strings.Split(content, "\n")
	if len(p.lines) > 0 && p.vOffset >= len(p.lines) {
		p.vOffset = len(p.lines) - p.height
		if p.vOffset < 0 {
			p.vOffset = 0
		}
	}
}

func (p *TextPreview) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *TextPreview) Len() int {
	if len(p.lines) == 0 || (len(p.lines) == 1 && p.lines[0] == "") {
		return 0
	}
	return len(p.lines)
}

func (p *TextPreview) ScrollUp(n int) {
	p.vOffset -= n
	if p.vOffset < 0 {
		p.vOffset = 0
	}
}

func (p *TextPreview) ScrollDown(n int) {
	p.vOffset += n
	p.clampOffset()
}

func (p *TextPreview) PageUp() {
	p.ScrollUp(p.height)
}

func (p *TextPreview) PageDown() {
	p.ScrollDown(p.height)
}

func (p *TextPreview) clampOffset() {
	max := len(p.lines) - p.height
	if max < 0 {
		max = 0
	}
	if p.vOffset > max {
		p.vOffset = max
	}
}

func (p *TextPreview) ScrollLeft(n int) {
	p.hOffset -= n
	if p.hOffset < 0 {
		p.hOffset = 0
	}
}

func (p *TextPreview) ScrollRight(n int) {
	p.hOffset += n
}

// DrawText renders the text preview content into a vaxis window.
// Println silently truncates at the window width, so gutter + content
// will be clipped naturally.
func (p *TextPreview) DrawText(win vaxis.Window) {
	if p.Len() == 0 {
		win.Println(0, vaxis.Segment{
			Text:  "(empty)",
			Style: emptyStyle,
		})
		return
	}

	end := p.vOffset + p.height
	if end > len(p.lines) {
		end = len(p.lines)
	}
	visible := p.lines[p.vOffset:end]

	for i, line := range visible {
		absN := p.vOffset + i + 1
		gutter := fmt.Sprintf("%6d | ", absN)

		if p.hOffset > 0 && len(line) > p.hOffset {
			line = line[p.hOffset:]
		}

		win.Println(i,
			vaxis.Segment{Text: gutter, Style: gutterStyle},
			vaxis.Segment{Text: line},
		)
	}
}

// VTPreview manages a term.Model for VT-rendered content preview.
type VTPreview struct {
	vt *term.Model
}

func NewVTPreview() *VTPreview {
	return &VTPreview{}
}

// Start begins a VT preview session, piping content through the given command.
// The term.Model forwards events to the provided eventFn (use vx.PostEvent).
func (vp *VTPreview) Start(content, cmdStr string, w, h int, eventFn func(vaxis.Event)) error {
	vp.Close()

	if cmdStr == "" {
		cmdStr = "cat"
	}

	vt := term.New()
	vt.TERM = "xterm-256color"
	vt.Attach(eventFn)

	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	if err := vt.StartWithSize(cmd, w, h); err != nil {
		return fmt.Errorf("start VT: %w", err)
	}

	// Write content and close stdin so the command can finish
	go func() {
		stdin.Write([]byte(content))
		stdin.Close()
	}()

	vp.vt = vt
	return nil
}

func (vp *VTPreview) Draw(win vaxis.Window) {
	if vp.vt != nil {
		w, h := win.Size()
		if w < 1 || h < 1 {
			return
		}
		vp.vt.Draw(win)
	}
}

func (vp *VTPreview) Resize(w, h int) {
	if vp.vt != nil {
		vp.vt.Resize(w, h)
	}
}

func (vp *VTPreview) Close() {
	if vp.vt != nil {
		vp.vt.Close()
		vp.vt = nil
	}
}
