package tui

import (
	"fmt"
	"os"
	"os/exec"

	"git.sr.ht/~rockorager/vaxis"
	vterm "git.sr.ht/~rockorager/vaxis/widgets/term"
	"github.com/szdytom/tb/internal/editor"
)

// EditorTab wraps a term.Model running $EDITOR on a temp file.
// The editor process runs in a PTY and is rendered via vaxis's terminal emulator.
type EditorTab struct {
	vt       *vterm.Model
	BufferID int64
	FilePath string // non-empty when editing a real file (e.g. config)
	TmpPath  string
	original string
	cmd      *exec.Cmd

	running bool
	done    bool
	closed  bool

	// Result (set after process exits)
	ExitCode      int
	ResultContent string
	ExitErr       error

	// Called when the editor process exits. Set by the App.
	onExit func()

	// Called for events from the terminal emulator (e.g. Redraw).
	// Set by the App to forward events to the main vaxis event loop.
	onEvent func(vaxis.Event)
}

// NewFileEditorTab creates an EditorTab that edits a real file in-place.
// The editor runs directly on the given path — no temp file involved.
// If the file does not exist, the editor will create it on save.
func NewFileEditorTab(filePath, editorStr string) (*EditorTab, error) {
	cmd := editor.BuildCmd(editorStr, filePath)

	vt := vterm.New()
	vt.TERM = "xterm-256color"
	vt.OSC8 = true

	return &EditorTab{
		FilePath: filePath,
		cmd:      cmd,
		vt:       vt,
	}, nil
}

func NewEditorTab(bufferID int64, content, editorStr string) (*EditorTab, error) {
	tmpPath, err := editor.CreateTempFile(content)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	cmd := editor.BuildCmd(editorStr, tmpPath)

	vt := vterm.New()
	vt.TERM = "xterm-256color"
	vt.OSC8 = true

	return &EditorTab{
		BufferID: bufferID,
		TmpPath:  tmpPath,
		original: content,
		cmd:      cmd,
		vt:       vt,
	}, nil
}

// Start launches the editor process in a PTY with the given dimensions.
// Uses deferred-start pattern: first Draw() with real dimensions calls this.
func (et *EditorTab) Start(w, h int) {
	if et.running || et.closed {
		return
	}

	// Attach callback before starting to ensure no events are missed
	et.vt.Attach(func(ev vaxis.Event) {
		switch e := ev.(type) {
		case vterm.EventClosed:
			if et.TmpPath != "" {
				content, err := editor.ReadFile(et.TmpPath)
				if err != nil {
					et.ExitErr = fmt.Errorf("read temp file: %w", err)
				} else {
					et.ResultContent = content
				}
			}

			if e.Error != nil {
				if exitErr, ok := e.Error.(*exec.ExitError); ok {
					et.ExitCode = exitErr.ExitCode()
				} else {
					et.ExitCode = -1
				}
			}

			et.done = true
			if et.onExit != nil {
				et.onExit()
			}
		default:
			// Forward Redraw and other events to the main vaxis event loop
			if et.onEvent != nil {
				et.onEvent(ev)
			}
		}
	})

	if err := et.vt.StartWithSize(et.cmd, w, h); err != nil {
		et.ExitErr = fmt.Errorf("start terminal: %w", err)

		et.done = true
		if et.onExit != nil {
			et.onExit()
		}

		return
	}

	et.running = true
}

func (et *EditorTab) Draw(win vaxis.Window) {
	if et.closed {
		return
	}

	w, h := win.Size()
	if w < 1 || h < 1 {
		return
	}

	if !et.running {
		et.Start(w, h)
	}

	et.vt.Draw(win)
}

func (et *EditorTab) HandleEvent(ev vaxis.Event) {
	if et.vt != nil && !et.closed {
		et.vt.Update(ev)
	}
}

func (et *EditorTab) Focus() {
	if et.vt != nil {
		et.vt.Focus()
	}
}

func (et *EditorTab) Blur() {
	if et.vt != nil {
		et.vt.Blur()
	}
}

func (et *EditorTab) Resize(w, h int) {
	if et.vt != nil && et.running {
		et.vt.Resize(w, h)
	}
}

func (et *EditorTab) Close() {
	if et.closed {
		return
	}

	et.closed = true
	et.running = false
	et.vt.Detach()
	et.vt.Close()
	if et.TmpPath != "" {
		os.Remove(et.TmpPath)
	}
}

func (et *EditorTab) Title() string {
	if et.FilePath != "" {
		return " config "
	}

	return fmt.Sprintf("EDIT:#%d", et.BufferID)
}
