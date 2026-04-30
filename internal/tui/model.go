package tui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/ipc"
)

// Client is the interface for daemon communication used by the TUI.
type Client interface {
	ListBufferSummaries(payload ipc.ListBuffersPayload) ([]buffer.BufferSummary, error)
	GetBuffer(id int64) (*buffer.Buffer, error)
	CreateBuffer(content, label string, tags []string) (int64, error)
	SoftDelete(id int64, ttlSeconds int) error
	Close() error
}

type state int

const (
	stateLoading state = iota
	stateBrowsing
	stateConfirmDelete
	stateHelp
	stateQuitting
)

// ── Messages ──────────────────────────────────────────────────────────

type buffersLoadedMsg struct {
	summaries []buffer.BufferSummary
	err       error
}

type bufferContentLoadedMsg struct {
	content string
	err     error
}

type bufferCreatedMsg struct {
	summary *buffer.BufferSummary
	err     error
}

type bufferDeletedMsg struct {
	id  int64
	err error
}

type errTimeoutMsg struct{}

// ── Styles ────────────────────────────────────────────────────────────

var (
	topBarStyle  = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	statusStyle  = lipgloss.NewStyle().Padding(0, 1)
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	confirmStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

// ── Model ─────────────────────────────────────────────────────────────

type Model struct {
	client Client

	state    state
	width    int
	height   int
	listW    int
	previewW int
	contentH int

	// List state
	summaries []buffer.BufferSummary
	cursor    int
	listOff   int

	// Preview
	preview *Preview

	awaitingColon bool
	deletingID    int64
	errMsg        string
	errFrames     int
}

func New(client Client) *Model {
	return &Model{
		client:    client,
		state:     stateLoading,
		preview:   NewPreview(),
		summaries: []buffer.BufferSummary{},
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadBuffers()
}

func (m Model) View() string {
	switch m.state {
	case stateLoading:
		return "Loading buffers..."
	case stateBrowsing, stateConfirmDelete:
		return m.mainView()
	case stateHelp:
		return m.mainView() + "\n" + HelpView(m.width, m.height)
	case stateQuitting:
		return ""
	}
	return ""
}

func (m Model) mainView() string {
	if !m.ready() {
		return ""
	}
	if m.contentH < 1 {
		return topBarStyle.Render(" tb - tmpbuffer ") + "\n" + statusStyle.Render(m.statusBarText())
	}

	// Render both panes to multi-line strings
	rawList := RenderList(m.summaries, m.cursor, m.listOff, m.contentH, m.listW)
	rawPreview := m.preview.View()

	// Split into lines and ensure each pane has exactly contentH lines
	listLines := splitPadded(rawList, m.contentH)
	previewLines := splitPadded(rawPreview, m.contentH)

	// Build the body: each line = left-pane | right-pane, padded to column widths
	div := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")
	var body strings.Builder
	for i := 0; i < m.contentH; i++ {
		if i > 0 {
			body.WriteByte('\n')
		}
		body.WriteString(padWidth(listLines[i], m.listW))
		body.WriteString(div)
		body.WriteString(padWidth(previewLines[i], m.previewW))
	}

	head := topBarStyle.Render(" tb - tmpbuffer ")
	status := statusStyle.Render(m.statusBarText())
	return head + "\n" + body.String() + "\n" + status
}

func (m Model) statusBarText() string {
	if m.state == stateConfirmDelete {
		return confirmStyle.Render(" Delete this buffer? (y/N) ")
	}
	if m.errFrames > 0 && m.errMsg != "" {
		return errStyle.Render(" " + m.errMsg + " ")
	}
	return " j/k:navigate  n:new  d:delete  ?:help  :q:quit "
}

func (m Model) ready() bool {
	return m.width > 0 && m.height > 0
}

// ── Helpers ───────────────────────────────────────────────────────────

// splitPadded splits a multi-line string into at most n lines.
// If there are fewer than n lines, empty lines are appended.
func splitPadded(s string, n int) []string {
	lines := strings.Split(s, "\n")
	if len(lines) >= n {
		return lines[:n]
	}
	padded := make([]string, n)
	copy(padded, lines)
	// remaining lines stay empty string
	return padded
}

// padWidth returns s padded with spaces on the right to reach width w,
// or truncated to w visible characters if longer.
func padWidth(s string, w int) string {
	vw := ansi.StringWidth(s)
	if vw >= w {
		// Truncate by visible width
		return ansi.Truncate(s, w, "")
	}
	return s + strings.Repeat(" ", w-vw)
}
