package tui

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
)

var (
	tabActiveStyle   = vaxis.Style{Attribute: vaxis.AttrReverse}
	tabInactiveStyle = vaxis.Style{}
	tabSepStyle      = vaxis.Style{Foreground: vaxis.IndexColor(240)}
)

// drawTabBar draws the tab bar at the top of the screen.
// Tab 0 is always the list tab. Editor tabs are tabs 1+.
func (a *App) drawTabBar(root vaxis.Window) {
	bar := root.New(0, 0, a.width, 1)

	x := 0
	x += a.drawTabItem(bar, x, "List", a.currentTab == 0)

	for i, tab := range a.editorTabs {
		title := tab.Title()
		x += a.drawTabSeparator(bar, x)
		x += a.drawTabItem(bar, x, title, a.currentTab == i+1)
	}
}

// drawTabSeparator draws a vertical separator between tabs.
func (a *App) drawTabSeparator(win vaxis.Window, x int) int {
	sep := win.New(x, 0, 1, 1)
	sep.SetCell(0, 0, vaxis.Cell{
		Character: vaxis.Character{Grapheme: "│", Width: 1},
		Style:     tabSepStyle,
	})

	return 1
}

// drawTabItem draws a single tab label at the given x offset within the tab bar.
// Returns the width consumed.
func (a *App) drawTabItem(win vaxis.Window, x int, label string, active bool) int {
	style := tabInactiveStyle
	if active {
		style = tabActiveStyle
	}

	text := fmt.Sprintf(" %s  ", label)
	item := win.New(x, 0, len(text), 1)
	item.PrintTruncate(0, vaxis.Segment{Text: text, Style: style})

	return len(text)
}
