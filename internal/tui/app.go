package tui

import (
	"fmt"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/textinput"
	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/ipc"
	"github.com/szdytom/tb/internal/store"
)

// ── Styles ────────────────────────────────────────────────────────────────

var (
	topBarStyle   = vaxis.Style{Attribute: vaxis.AttrBold}
	errStyle      = vaxis.Style{Foreground: vaxis.IndexColor(9)}
	confirmStyle  = vaxis.Style{Foreground: vaxis.IndexColor(11)}
	selectedStyle = vaxis.Style{Attribute: vaxis.AttrReverse}
	headerStyle   = vaxis.Style{Attribute: vaxis.AttrBold}
	emptyStyle    = vaxis.Style{Foreground: vaxis.IndexColor(240), Attribute: vaxis.AttrItalic}
	gutterStyle   = vaxis.Style{Foreground: vaxis.IndexColor(240)}
	divStyle      = vaxis.Style{Foreground: vaxis.IndexColor(240)}
)

// ── TUI App ──────────────────────────────────────────────────────────────

type state int

const (
	stateLoading state = iota
	stateBrowsing
	stateConfirmDelete
	stateSearch
	stateHelp
	stateQuitting
)

// Client is the interface for daemon communication used by the TUI.
type Client interface {
	ListBufferSummaries(payload ipc.ListBuffersPayload) ([]buffer.BufferSummary, error)
	GetBuffer(id int64) (*buffer.Buffer, error)
	CreateBuffer(content, label string, tags []string) (int64, error)
	SoftDelete(id int64, ttlSeconds int) error
	Search(query string, isRegex bool) ([]store.SearchResult, error)
	Close() error
}

// App holds all state for the vaxis-based TUI.
type App struct {
	vx     *vaxis.Vaxis
	client Client

	width, height   int
	listW, previewW int
	contentH        int

	curState state

	// List state
	summaries []buffer.BufferSummary
	cursor    int
	listOff   int

	// Preview
	textPreview *TextPreview
	vtPreview   *VTPreview
	vtActive    bool

	// Stale load guard: increment each time a new preview is requested;
	// async completion checks against current generation.
	previewGen int

	// Search state
	allSummaries      []buffer.BufferSummary
	searchInput       *textinput.Model
	searchQuery       string // saved after commit, shown in status bar
	searchTimer       *time.Timer
	searchGen         int
	savedSearchQuery  string // filter that was active when search was entered
	savedSearchFilter []buffer.BufferSummary // summaries when search was entered

	// Confirm delete
	deletingID int64

	// Preview command (e.g. "bat --color=always --style=plain")
	previewCmd string

	// Misc
	awaitingColon bool
	errMsg        string
	errTimer      *time.Timer
	quitting      bool
}

func New(client Client, previewCmd string) *App {
	return &App{
		client:      client,
		previewCmd:  previewCmd,
		textPreview: NewTextPreview(),
		vtPreview:   NewVTPreview(),
		summaries:   []buffer.BufferSummary{},
		curState:    stateLoading,
	}
}

// Run initializes vaxis and enters the event loop. This blocks until the TUI
// is closed by the user.
func (a *App) Run() error {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		return fmt.Errorf("vaxis init: %w", err)
	}
	defer vx.Close()

	a.vx = vx

	root := vx.Window()
	w, h := root.Size()
	a.width = w
	a.height = h
	a.recalcLayout()

	vx.HideCursor()

	// Kick off initial buffer load
	a.loadBuffersAsync()

	for ev := range vx.Events() {
		a.handleEvent(ev)
		if a.quitting {
			break
		}
		a.draw()
		vx.Render()
	}

	// Cleanup
	a.vtPreview.Close()

	return nil
}

// ── Layout calculations ───────────────────────────────────────────────────

func (a *App) recalcLayout() {
	a.listW = a.width * 40 / 100
	if a.listW < 30 {
		a.listW = 30
	}
	// Ensure listW doesn't consume the whole width; leave room for divider + preview
	maxList := a.width - 2
	if a.listW > maxList {
		a.listW = maxList
	}
	a.previewW = a.width - a.listW - 1
	if a.previewW < 1 {
		a.previewW = 1
	}
	a.contentH = a.height - 2
	if a.contentH < 1 {
		a.contentH = 1
	}
	a.textPreview.SetSize(a.previewW, a.contentH)
	if a.vtActive {
		a.vtPreview.Resize(a.previewW, a.contentH)
	}
}

// ── Draw ──────────────────────────────────────────────────────────────────

func (a *App) draw() {
	root := a.vx.Window()
	root.Clear()

	switch a.curState {
	case stateLoading:
		a.drawLoading(root)
	case stateBrowsing, stateSearch, stateConfirmDelete, stateHelp:
		a.drawMainView(root)
		if a.curState == stateHelp {
			DrawHelp(root, a.width, a.contentH)
		}
	case stateQuitting:
		// Nothing to draw
	}
}

func (a *App) drawLoading(root vaxis.Window) {
	row := a.height / 2
	if row < 0 {
		row = 0
	}
	win := root.New(0, row, a.width, 1)
	win.Println(0, vaxis.Segment{Text: "Loading buffers..."})
}

func (a *App) drawMainView(root vaxis.Window) {
	// Top bar
	topBar := root.New(0, 0, a.width, 1)
	topBar.PrintTruncate(0, vaxis.Segment{
		Text:  " tb - tmpbuffer ",
		Style: topBarStyle,
	})

	// List pane
	if a.contentH > 0 {
		listWin := root.New(0, 1, a.listW, a.contentH)
		DrawList(listWin, a.summaries, a.cursor, a.listOff, a.contentH)
	}

	// Divider
	divWin := root.New(a.listW, 1, 1, a.contentH)
	wDiv, hDiv := divWin.Size()
	for y := 0; y < hDiv && y < wDiv; y++ {
		divWin.SetCell(0, y, vaxis.Cell{
			Character: vaxis.Character{Grapheme: "│", Width: 1},
			Style:     divStyle,
		})
	}

	// Preview pane
	if a.contentH > 0 {
		prevWin := root.New(a.listW+1, 1, a.previewW, a.contentH)
		if a.vtActive {
			a.vtPreview.Draw(prevWin)
		} else {
			a.textPreview.DrawText(prevWin)
		}
	}

	// Status bar
	statusWin := root.New(0, a.height-1, a.width, 1)
	var text string
	var style vaxis.Style
	switch {
	case a.curState == stateConfirmDelete:
		text = " Delete this buffer? (y/N) "
		style = confirmStyle
	case a.curState == stateSearch:
		a.drawSearchBar(statusWin)
		return
	case a.errMsg != "":
		text = " " + a.errMsg + " "
		style = errStyle
	case a.searchQuery != "":
		text = a.searchStatusText()
	default:
		text = " j/k:navigate  n:new  d:delete  ?:help  :q:quit "
	}
	statusWin.PrintTruncate(0, vaxis.Segment{Text: text, Style: style})
}

// ── IPC: Async buffer operations ──────────────────────────────────────────

func (a *App) loadBuffersAsync() {
	go func() {
		summaries, err := a.client.ListBufferSummaries(ipc.ListBuffersPayload{
			SortBy:  string(store.SortByUpdatedAt),
			SortAsc: false,
		})
		a.vx.PostEvent(buffersLoaded{summaries: summaries, err: err})
	}()
}

func (a *App) loadPreviewAsync() {
	if len(a.summaries) == 0 {
		return
	}
	id := a.summaries[a.cursor].ID
	a.previewGen++

	gen := a.previewGen
	go func() {
		buf, err := a.client.GetBuffer(id)
		a.vx.PostEvent(contentLoaded{
			id:      id,
			gen:     gen,
			content: buf,
			err:     err,
		})
	}()
}

func (a *App) createBufferAsync() {
	go func() {
		id, err := a.client.CreateBuffer("", "", nil)
		if err != nil {
			a.vx.PostEvent(bufferCreated{err: err})
			return
		}
		buf, err := a.client.GetBuffer(id)
		if err != nil {
			a.vx.PostEvent(bufferCreated{err: err})
			return
		}
		s := buffer.NewBufferSummary(buf)
		a.vx.PostEvent(bufferCreated{summary: &s})
	}()
}

func (a *App) deleteBufferAsync(id int64) {
	go func() {
		err := a.client.SoftDelete(id, 86400)
		a.vx.PostEvent(bufferDeleted{id: id, err: err})
	}()
}

// ── VT Preview helpers ────────────────────────────────────────────────────

func (a *App) startVTPreview(content string) {
	a.vtPreview.Close()
	a.vtActive = false

	w := a.previewW
	h := a.contentH
	if w < 1 || h < 1 {
		return
	}

	cmdStr := a.previewCmd
	if cmdStr == "" {
		cmdStr = "cat"
	}
	err := a.vtPreview.Start(
		content,
		cmdStr,
		w, h,
		func(ev vaxis.Event) { a.vx.PostEvent(ev) },
	)
	if err != nil {
		// Fall through to text preview
		return
	}
	a.vtActive = true
}

// ── Error handling ────────────────────────────────────────────────────────

func (a *App) setError(msg string) {
	a.errMsg = msg
	if a.errTimer != nil {
		a.errTimer.Stop()
	}
	a.errTimer = time.AfterFunc(2*time.Second, func() {
		a.vx.PostEvent(errClear{})
	})
}

// ── State helpers ─────────────────────────────────────────────────────────

func (a *App) moveDown() {
	if a.cursor < len(a.summaries)-1 {
		a.cursor++
		a.clampListOff()
	}
}

func (a *App) moveUp() {
	if a.cursor > 0 {
		a.cursor--
		a.clampListOff()
	}
}

func (a *App) clampListOff() {
	if a.cursor < a.listOff {
		a.listOff = a.cursor
	}
	if a.listOff > a.cursor {
		a.listOff = a.cursor
	}
	maxOff := a.cursor - a.contentH + 3
	if a.cursor >= a.listOff+a.contentH-2 && a.contentH > 0 && maxOff > a.listOff {
		a.listOff = maxOff
	}
}

func (a *App) clampCursor() {
	if a.cursor < 0 {
		a.cursor = 0
	}
	if len(a.summaries) > 0 && a.cursor >= len(a.summaries) {
		a.cursor = len(a.summaries) - 1
	}
}
