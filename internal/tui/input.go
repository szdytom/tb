package tui

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/textinput"
)

type inputAction int

const (
	inputNone inputAction = iota
	inputCommit
	inputCancel
	inputChanged
)

// statusInput wraps textinput.Model for use as a status bar input widget.
// It handles the common key event lifecycle for search (/), command (:),
// and other status-bar input modes.
type statusInput struct {
	model *textinput.Model
}

func newStatusInput(prompt string) *statusInput {
	m := textinput.New()
	m.SetPrompt(prompt)

	return &statusInput{model: m}
}

func (s *statusInput) Draw(win vaxis.Window) {
	s.model.Draw(win)
}

func (s *statusInput) String() string {
	return s.model.String()
}

// Update forwards non-key events (e.g., PasteEndEvent) to the underlying model.
func (s *statusInput) Update(ev vaxis.Event) {
	s.model.Update(ev)
}

// HandleKey processes a key event and returns an action along with the current
// input text. The caller should switch on the action:
//   - inputCommit:  Enter was pressed (value holds the full text)
//   - inputCancel:  Escape, Ctrl-c, or Backspace on empty (value is empty)
//   - inputChanged: text content changed (value holds new text)
//   - inputNone:    key was consumed but nothing of note happened
func (s *statusInput) HandleKey(ev vaxis.Key) (inputAction, string) {
	switch {
	case ev.Matches(vaxis.KeyEsc):
		return inputCancel, ""
	case ev.Matches('c', vaxis.ModCtrl):
		return inputCancel, ""
	case ev.Matches(vaxis.KeyEnter):
		return inputCommit, s.model.String()
	}

	if s.model.String() == "" && (ev.String() == "BackSpace" || ev.String() == "Ctrl+h") {
		return inputCancel, ""
	}

	old := s.model.String()
	s.model.Update(ev)
	cur := s.model.String()

	if cur != old {
		return inputChanged, cur
	}

	return inputNone, cur
}
