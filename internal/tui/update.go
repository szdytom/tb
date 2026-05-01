package tui

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
	vterm "git.sr.ht/~rockorager/vaxis/widgets/term"
	"github.com/szdytom/tb/internal/buffer"
)

// ── Custom event types ────────────────────────────────────────────────────

type buffersLoaded struct {
	summaries []buffer.BufferSummary
	err       error
}

type contentLoaded struct {
	id      int64
	gen     int
	content *buffer.Buffer
	err     error
}

type bufferCreated struct {
	summary *buffer.BufferSummary
	err     error
}

type bufferDeleted struct {
	id  int64
	err error
}

type errClear struct{}

type editorStarted struct {
	tab *EditorTab
	err error
}

type editorExited struct {
	tab *EditorTab
	err error
}

// ── Event dispatch ──────────────────────────────────────────────────────

func (a *App) handleEvent(ev vaxis.Event) {
	switch ev := ev.(type) {
	case vaxis.Key:
		a.handleKey(ev)
	case vaxis.Mouse:
		a.handleMouse(ev)
	case vaxis.Resize:
		a.handleResize(ev)
	case buffersLoaded:
		a.handleBuffersLoaded(ev)
	case contentLoaded:
		a.handleContentLoaded(ev)
	case bufferCreated:
		a.handleBufferCreated(ev)
	case bufferDeleted:
		a.handleBufferDeleted(ev)
	case searchResult:
		a.handleSearchResult(ev)
	case editorStarted:
		a.handleEditorStarted(ev)
	case editorExited:
		a.handleEditorExited(ev)
	case errClear:
		a.errMsg = ""
	case vterm.EventClosed:
		// VT preview or editor tab finished; handled via editorExited for editor tabs
	case vaxis.Redraw:
		// Triggered after SyncFunc; state already updated, just re-draw
	case vaxis.PasteEndEvent:
		if a.curState == stateSearch && a.searchInput != nil {
			a.searchInput.Update(ev)
			a.searchDebounced()
		}
	}
}

// ── Resize ───────────────────────────────────────────────────────────────

func (a *App) handleResize(ev vaxis.Resize) {
	a.width = ev.Cols
	a.height = ev.Rows
	a.recalcLayout()
}

func (a *App) handleMouse(ev vaxis.Mouse) {
	// Tab bar click (row 0)
	if ev.Row == 0 && (a.curState == stateBrowsing || a.curState == stateHelp) {
		if tabIdx := a.tabAtX(ev.Col); tabIdx >= 0 {
			a.currentTab = tabIdx
			a.updateTabFocus()
		}
		return
	}

	// Forward to the editor tab if on an editor tab
	if a.currentTab > 0 && a.curState == stateBrowsing {
		tab := a.editorTabs[a.currentTab-1]
		e := ev
		e.Row = ev.Row - 1 // tab bar offset
		tab.HandleEvent(e)
	}
}

// ── Key dispatch ─────────────────────────────────────────────────────────

func (a *App) handleLeaderKey(ev vaxis.Key) bool {
	const leader = "Ctrl+b"

	if a.leaderPending {
		a.leaderPending = false
		switch ev.String() {
		case leader:
			// Leader pressed twice → pass through to the editor
			return false
		case "1":
			a.currentTab = 0
			a.updateTabFocus()
			return true
		case "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(ev.String()[0] - '1') // "2" → 1 (first editor tab)
			if idx <= len(a.editorTabs) {
				a.currentTab = idx
				a.updateTabFocus()
			}
			return true
		case "n":
			a.createBufferAsync()
			return true
		case "q":
			for _, tab := range a.editorTabs {
				tab.Close()
			}
			a.editorTabs = nil
			a.quitting = true
			return true
		case "?":
			a.curState = stateHelp
			return true
		default:
			// Unknown key: cancel leader, forward to editor
			return false
		}
	}

	if ev.String() == leader && len(a.editorTabs) > 0 {
		a.leaderPending = true
		return true
	}

	return false
}

func (a *App) handleKey(ev vaxis.Key) {
	if a.awaitingColon && a.currentTab == 0 {
		a.awaitingColon = false
		if ev.String() == "q" {
			// Close all editor tabs before quitting
			for _, tab := range a.editorTabs {
				tab.Close()
			}
			a.editorTabs = nil
			a.quitting = true
		}
		return
	}

	// Leader key works from both list and editor tabs
	if a.handleLeaderKey(ev) {
		return
	}

	switch a.curState {
	case stateBrowsing:
		if a.currentTab > 0 {
			tab := a.editorTabs[a.currentTab-1]
			tab.HandleEvent(ev)
			return
		}
		a.handleKeyBrowsing(ev)
	case stateSearch:
		a.handleKeySearch(ev)
	case stateConfirmDelete:
		a.handleKeyConfirmDelete(ev)
	case stateEditorExitConfirm:
		a.handleKeyEditorExitConfirm(ev)
	case stateHelp:
		if ev.String() == "?" || ev.String() == "Escape" {
			a.curState = stateBrowsing
		}
	}
}

func (a *App) handleKeyBrowsing(ev vaxis.Key) {
	if ev.String() == ":" {
		a.awaitingColon = true
		return
	}

	// Escape/Ctrl+c while a search filter is active → clear the filter
	if a.searchQuery != "" && (ev.Matches(vaxis.KeyEsc) || ev.Matches('c', vaxis.ModCtrl) || ev.String() == "Escape") {
		a.clearFilter()
		return
	}

	switch classifyKey(ev) {
	case keyDown:
		a.moveDown()
		a.loadPreviewAsync()
	case keyUp:
		a.moveUp()
		a.loadPreviewAsync()
	case keyPageDown:
		step := a.contentH - 1
		if step < 1 {
			step = 1
		}
		a.cursor += step
		a.clampCursor()
		a.listOff = a.cursor
		a.loadPreviewAsync()
	case keyPageUp:
		step := a.contentH - 1
		if step < 1 {
			step = 1
		}
		a.cursor -= step
		a.clampCursor()
		a.listOff = a.cursor
		a.loadPreviewAsync()
	case keyHome:
		a.cursor = 0
		a.listOff = 0
		a.loadPreviewAsync()
	case keyEnd:
		a.cursor = len(a.summaries) - 1
		if a.cursor < 0 {
			a.cursor = 0
		}
		a.listOff = a.cursor - a.contentH + 3
		if a.listOff < 0 {
			a.listOff = 0
		}
		a.loadPreviewAsync()
	case keyEnter:
		a.startEditorAsync()
	case keyNew:
		a.createBufferAsync()
	case keyDelete:
		if len(a.summaries) > 0 {
			a.deletingID = a.summaries[a.cursor].ID
			a.curState = stateConfirmDelete
		}
	case keyHelp:
		a.curState = stateHelp
	case keySearch:
		a.enterSearch()
	case keyQuit:
		a.quitting = true
	}
}

func (a *App) handleKeyConfirmDelete(ev vaxis.Key) {
	switch classifyKey(ev) {
	case keyConfirm:
		a.curState = stateBrowsing
		a.deleteBufferAsync(a.deletingID)
	case keyDeny:
		a.curState = stateBrowsing
		a.deletingID = 0
	}
}

// ── Internal message handlers ───────────────────────────────────────────

func (a *App) handleBuffersLoaded(msg buffersLoaded) {
	if msg.err != nil {
		a.setError(fmt.Sprintf("Failed to load buffers: %v", msg.err))
		a.curState = stateBrowsing
		return
	}
	a.summaries = msg.summaries
	a.allSummaries = msg.summaries
	a.curState = stateBrowsing
	if len(msg.summaries) > 0 {
		a.loadPreviewAsync()
	}
}

func (a *App) handleContentLoaded(msg contentLoaded) {
	if msg.err != nil {
		a.setError(fmt.Sprintf("Failed to load preview: %v", msg.err))
		return
	}
	// Discard stale loads from older navigation
	if msg.gen != a.previewGen {
		return
	}

	a.textPreview.SetContent(msg.content.Content)

	// Use VT preview if a preview command is configured, or if content has ANSI escapes
	useVT := a.previewCmd != ""
	if !useVT {
		content := msg.content.Content
		for i := 0; i < len(content)-1; i++ {
			if content[i] == '\x1b' && content[i+1] == '[' {
				useVT = true
				break
			}
		}
	}
	if useVT && a.previewW > 0 && a.contentH > 0 {
		a.startVTPreview(msg.content.Content)
	} else {
		a.vtActive = false
	}
}

func (a *App) handleBufferCreated(msg bufferCreated) {
	if msg.err != nil {
		a.setError(fmt.Sprintf("Failed to create buffer: %v", msg.err))
		return
	}
	if msg.summary != nil {
		summary := *msg.summary
		a.allSummaries = append([]buffer.BufferSummary{summary}, a.allSummaries...)
		if a.curState == stateSearch {
			a.triggerSearch()
		} else {
			a.summaries = append([]buffer.BufferSummary{summary}, a.summaries...)
			a.cursor = 0
			a.listOff = 0
			a.vtPreview.Close()
			a.vtActive = false
			a.loadPreviewAsync()
		}
	}
}

func (a *App) handleBufferDeleted(msg bufferDeleted) {
	if msg.err != nil {
		a.setError(fmt.Sprintf("Failed to delete buffer: %v", msg.err))
		return
	}

	removeFromList := func(sl *[]buffer.BufferSummary, id int64) {
		for i, s := range *sl {
			if s.ID == id {
				*sl = append((*sl)[:i], (*sl)[i+1:]...)
				break
			}
		}
	}

	removeFromList(&a.allSummaries, msg.id)

	if a.curState == stateSearch {
		a.triggerSearch()
	} else {
		removeFromList(&a.summaries, msg.id)
		if a.cursor >= len(a.summaries) && a.cursor > 0 {
			a.cursor--
		}
		if a.listOff > a.cursor {
			a.listOff = a.cursor
		}
	}

	a.deletingID = 0
	if len(a.summaries) > 0 {
		a.loadPreviewAsync()
	}
}

// ── Editor lifecycle handlers ────────────────────────────────────────

func (a *App) handleKeyEditorExitConfirm(ev vaxis.Key) {
	switch ev.String() {
	case "y", "Y":
		// Keep changes
		idx := a.confirmExitTabIdx
		if idx >= 0 && idx < len(a.editorTabs) {
			tab := a.editorTabs[idx]
			if err := a.client.UpdateContent(tab.BufferID, tab.ResultContent); err != nil {
				a.setError("Failed to save editor changes: " + err.Error())
			}
		}
		a.closeEditorTab(a.confirmExitTabIdx)
		a.curState = stateBrowsing
	case "n", "N", "Escape":
		// Discard changes
		a.closeEditorTab(a.confirmExitTabIdx)
		a.curState = stateBrowsing
	}
}

func (a *App) handleEditorStarted(msg editorStarted) {
	if msg.err != nil {
		a.setError("Failed to start editor: " + msg.err.Error())
		return
	}

	tab := msg.tab
	tabIdx := len(a.editorTabs)

	// Set up callbacks
	tab.onExit = func() {
		a.vx.PostEvent(editorExited{tab: tab, err: nil})
	}
	// Forward Redraw/other events from term.Model to the main vaxis loop
	tab.onEvent = func(ev vaxis.Event) {
		a.vx.PostEvent(ev)
	}

	a.editorTabs = append(a.editorTabs, tab)
	a.currentTab = tabIdx + 1 // 1-based: 1 = first editor tab
	tab.Focus()
}

func (a *App) handleEditorExited(msg editorExited) {
	tab := msg.tab

	// Find the tab index
	tabIdx := -1
	for i, t := range a.editorTabs {
		if t == tab {
			tabIdx = i
			break
		}
	}
	if tabIdx < 0 {
		return // Already handled
	}

	// If the tab was closed while the editor was running, clean up
	if tab.closed {
		return
	}

	// Check exit code
	if tab.ExitCode != 0 && !tab.closed {
		// Non-zero: prompt user
		a.confirmExitTabIdx = tabIdx
		a.confirmExitCode = tab.ExitCode
		a.curState = stateEditorExitConfirm
		// Switch to the editor tab if not already there
		if a.currentTab != tabIdx+1 {
			a.currentTab = tabIdx + 1
			a.updateTabFocus()
		}
		return
	}

	// Zero exit (or no exit code): auto-save
	if tab.ResultContent != "" && tab.ResultContent != tab.original {
		if err := a.client.UpdateContent(tab.BufferID, tab.ResultContent); err != nil {
			a.setError("Failed to save editor changes: " + err.Error())
		}
	}
	a.closeEditorTab(tabIdx)
}

// closeEditorTab removes an editor tab at the given index, cleaning up resources.
func (a *App) closeEditorTab(idx int) {
	if idx < 0 || idx >= len(a.editorTabs) {
		return
	}

	tab := a.editorTabs[idx]
	tab.Close()

	// Remove from slice
	a.editorTabs = append(a.editorTabs[:idx], a.editorTabs[idx+1:]...)

	// Adjust current tab
	if len(a.editorTabs) == 0 {
		a.currentTab = 0
	} else if a.currentTab > idx+1 {
		a.currentTab--
	} else if a.currentTab > len(a.editorTabs) {
		a.currentTab = len(a.editorTabs)
	}

	a.updateTabFocus()

	// Reload summaries to reflect any content change
	a.loadBuffersAsync()
}
