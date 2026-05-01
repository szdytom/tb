package tui

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
	"github.com/szdytom/tb/internal/buffer"
	vterm "git.sr.ht/~rockorager/vaxis/widgets/term"
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

// ── Event dispatch ──────────────────────────────────────────────────────

func (a *App) handleEvent(ev vaxis.Event) {
	switch ev := ev.(type) {
	case vaxis.Key:
		a.handleKey(ev)
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
	case errClear:
		a.errMsg = ""
	case vterm.EventClosed:
		// VT preview command finished; nothing special needed
	case vaxis.Redraw:
		// Triggered after SyncFunc; state already updated, just re-draw
	}
}

// ── Resize ───────────────────────────────────────────────────────────────

func (a *App) handleResize(ev vaxis.Resize) {
	a.width = ev.Cols
	a.height = ev.Rows
	a.recalcLayout()
}

// ── Key dispatch ─────────────────────────────────────────────────────────

func (a *App) handleKey(ev vaxis.Key) {
	if a.awaitingColon {
		a.awaitingColon = false
		if ev.String() == "q" {
			a.quitting = true
		}
		return
	}

	switch a.curState {
	case stateBrowsing:
		a.handleKeyBrowsing(ev)
	case stateConfirmDelete:
		a.handleKeyConfirmDelete(ev)
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
	case keyNew:
		a.createBufferAsync()
	case keyDelete:
		if len(a.summaries) > 0 {
			a.deletingID = a.summaries[a.cursor].ID
			a.curState = stateConfirmDelete
		}
	case keyHelp:
		a.curState = stateHelp
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
		a.summaries = append([]buffer.BufferSummary{*msg.summary}, a.summaries...)
		a.cursor = 0
		a.listOff = 0
		a.vtPreview.Close()
		a.vtActive = false
		a.loadPreviewAsync()
	}
}

func (a *App) handleBufferDeleted(msg bufferDeleted) {
	if msg.err != nil {
		a.setError(fmt.Sprintf("Failed to delete buffer: %v", msg.err))
		return
	}
	for i, s := range a.summaries {
		if s.ID == msg.id {
			a.summaries = append(a.summaries[:i], a.summaries[i+1:]...)
			if a.cursor >= len(a.summaries) && a.cursor > 0 {
				a.cursor--
			}
			if a.listOff > a.cursor {
				a.listOff = a.cursor
			}
			break
		}
	}
	a.deletingID = 0
	// Reload preview if we still have buffers
	if len(a.summaries) > 0 {
		a.loadPreviewAsync()
	}
}
