package tui

import (
	"git.sr.ht/~rockorager/vaxis"
)

func DrawHelp(root vaxis.Window, screenW, screenH int) {
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

	contentW := 0
	for _, l := range lines {
		if len(l) > contentW {
			contentW = len(l)
		}
	}
	pad := 2
	boxW := contentW + pad*2 + 2
	boxH := len(lines) + pad*2 + 2

	x := (screenW - boxW) / 2
	y := (screenH - boxH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	box := root.New(x, y, boxW, boxH)
	box.Fill(vaxis.Cell{
		Character: vaxis.Character{Grapheme: " ", Width: 1},
	})

	drawBoxBorder(box, vaxis.Style{Attribute: vaxis.AttrBold})

	for i, line := range lines {
		box.PrintTruncate(pad+1+i, vaxis.Segment{Text: line})
	}
}

func drawBoxBorder(win vaxis.Window, style vaxis.Style) {
	w, h := win.Size()
	win.SetCell(0, 0, cell('┌', style))
	win.SetCell(w-1, 0, cell('┐', style))
	win.SetCell(0, h-1, cell('└', style))
	win.SetCell(w-1, h-1, cell('┘', style))
	for x := 1; x < w-1; x++ {
		win.SetCell(x, 0, cell('─', style))
		win.SetCell(x, h-1, cell('─', style))
	}
	for y := 1; y < h-1; y++ {
		win.SetCell(0, y, cell('│', style))
		win.SetCell(w-1, y, cell('│', style))
	}
}

func cell(grapheme rune, style vaxis.Style) vaxis.Cell {
	return vaxis.Cell{
		Character: vaxis.Character{Grapheme: string(grapheme), Width: 1},
		Style:     style,
	}
}
