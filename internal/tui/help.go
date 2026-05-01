package tui

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/border"
)

func DrawHelp(root vaxis.Window, screenW, screenH int) {
	lines := []string{
		" j / Up          Move selection up",
		" k / Down        Move selection down",
		" PgUp / PgDn     Page up / down",
		" g / G           Top / bottom",
		" Enter           Open in editor",
		" n               New buffer",
		" d               Delete buffer",
		" /               Search",
		" C-b <n>         Switch tab (1:list, 2-9:editor)",
		"   C-b 1         List tab",
		"   C-b 2-9       Editor tabs",
		"   C-b n         New buffer",
		"   C-b q         Quit",
		" :q / Ctrl+C     Quit",
		" ?               Toggle help",
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
	inner := border.All(box, vaxis.Style{Attribute: vaxis.AttrBold})
	for i, line := range lines {
		inner.Println(i+pad, vaxis.Segment{Text: line})
	}
}
