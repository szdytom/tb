package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/ipc"
	"github.com/szdytom/tb/internal/store"
)

// ── Update ────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg), nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case buffersLoadedMsg:
		return m.handleBuffersLoaded(msg)
	case bufferContentLoadedMsg:
		return m.handleContentLoaded(msg), nil
	case bufferCreatedMsg:
		return m.handleBufferCreated(msg), nil
	case bufferDeletedMsg:
		return m.handleBufferDeleted(msg), nil
	case errTimeoutMsg:
		m.errMsg = ""
		m.errFrames = 0
		return m, nil
	}
	return m, nil
}

// ── Resize ────────────────────────────────────────────────────────────

func (m Model) handleResize(msg tea.WindowSizeMsg) Model {
	m.width = msg.Width
	m.height = msg.Height

	m.listW = m.width * 40 / 100
	if m.listW < 30 {
		m.listW = 30
	}
	m.previewW = m.width - m.listW - 1
	if m.previewW < 20 {
		m.previewW = 20
	}

	contentH := m.height - 2
	if contentH < 1 {
		contentH = 1
	}
	m.contentH = contentH
	m.preview.SetSize(m.previewW, m.contentH)
	return m
}

// ── Key handlers ──────────────────────────────────────────────────────

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.awaitingColon {
		m.awaitingColon = false
		if msg.String() == "q" {
			m.state = stateQuitting
			return m, tea.Quit
		}
	}

	switch m.state {
	case stateBrowsing:
		return m.handleKeyBrowsing(msg)
	case stateConfirmDelete:
		return m.handleKeyConfirmDelete(msg)
	case stateHelp:
		if msg.String() == "?" || msg.String() == "esc" {
			m.state = stateBrowsing
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKeyBrowsing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == ":" {
		m.awaitingColon = true
		return m, nil
	}

	switch classifyKey(msg) {
	case keyDown:
		m.moveDown()
		return m, m.loadPreview()
	case keyUp:
		m.moveUp()
		return m, m.loadPreview()
	case keyPageDown:
		step := m.contentH - 1; if step < 1 { step = 1 }; m.cursor += step
		if m.cursor >= len(m.summaries) {
			m.cursor = len(m.summaries) - 1
		}
		m.listOff = m.cursor
		return m, m.loadPreview()
	case keyPageUp:
		step := m.contentH - 1; if step < 1 { step = 1 }; m.cursor -= step
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.listOff = m.cursor
		return m, m.loadPreview()
	case keyHome:
		m.cursor = 0
		m.listOff = 0
		return m, m.loadPreview()
	case keyEnd:
		m.cursor = len(m.summaries) - 1
		m.listOff = m.cursor - m.contentH + 3
		if m.listOff < 0 {
			m.listOff = 0
		}
		return m, m.loadPreview()
	case keyNew:
		return m, m.createBuffer()
	case keyDelete:
		if len(m.summaries) > 0 {
			m.deletingID = m.summaries[m.cursor].ID
			m.state = stateConfirmDelete
		}
		return m, nil
	case keyHelp:
		m.state = stateHelp
		return m, nil
	case keyQuit:
		m.state = stateQuitting
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleKeyConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch classifyKey(msg) {
	case keyConfirm:
		m.state = stateBrowsing
		return m, m.deleteBuffer(m.deletingID)
	case keyDeny:
		m.state = stateBrowsing
		m.deletingID = 0
		return m, nil
	}
	return m, nil
}

func (m *Model) moveDown() {
	if m.cursor < len(m.summaries)-1 {
		m.cursor++
		m.clampListOff()
	}
}

func (m *Model) moveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.clampListOff()
	}
}

func (m *Model) clampListOff() {
	if m.cursor < m.listOff {
		m.listOff = m.cursor
	}
	if m.cursor >= m.listOff+m.contentH-2 && m.contentH > 0 {
		m.listOff = m.cursor - m.contentH + 3
		if m.listOff < 0 {
			m.listOff = 0
		}
	}
}

// ── IPC commands ──────────────────────────────────────────────────────

func (m Model) loadBuffers() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		summaries, err := client.ListBufferSummaries(ipc.ListBuffersPayload{
			SortBy:  string(store.SortByUpdatedAt),
			SortAsc: false,
		})
		return buffersLoadedMsg{summaries: summaries, err: err}
	}
}

func (m Model) loadPreview() tea.Cmd {
	if len(m.summaries) == 0 {
		return nil
	}
	client := m.client
	id := m.summaries[m.cursor].ID
	return func() tea.Msg {
		buf, err := client.GetBuffer(id)
		if err != nil {
			return bufferContentLoadedMsg{err: err}
		}
		return bufferContentLoadedMsg{content: buf.Content}
	}
}

func (m Model) createBuffer() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		id, err := client.CreateBuffer("", "", nil)
		if err != nil {
			return bufferCreatedMsg{err: err}
		}
		buf, err := client.GetBuffer(id)
		if err != nil {
			return bufferCreatedMsg{err: err}
		}
		s := buffer.NewBufferSummary(buf)
		return bufferCreatedMsg{summary: &s}
	}
}

func (m Model) deleteBuffer(id int64) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		err := client.SoftDelete(id, 86400)
		return bufferDeletedMsg{id: id, err: err}
	}
}

// ── Message handlers ──────────────────────────────────────────────────

func (m Model) handleBuffersLoaded(msg buffersLoadedMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = fmt.Sprintf("Failed to load buffers: %v", msg.err)
		m.errFrames = 60
		m.state = stateBrowsing
		return m, nil
	}
	m.summaries = msg.summaries
	m.state = stateBrowsing
	// Trigger preview load for first buffer
	if len(m.summaries) > 0 {
		return m, m.loadPreview()
	}
	return m, nil
}

func (m Model) handleContentLoaded(msg bufferContentLoadedMsg) Model {
	if msg.err != nil {
		m.errMsg = fmt.Sprintf("Failed to load preview: %v", msg.err)
		m.errFrames = 60
		return m
	}
	m.preview.SetContent(msg.content)
	m.preview.SetSize(m.previewW, m.contentH)
	return m
}

func (m Model) handleBufferCreated(msg bufferCreatedMsg) Model {
	if msg.err != nil {
		m.errMsg = fmt.Sprintf("Failed to create buffer: %v", msg.err)
		m.errFrames = 60
		return m
	}
	if msg.summary != nil {
		m.summaries = append([]buffer.BufferSummary{*msg.summary}, m.summaries...)
		m.cursor = 0
		m.listOff = 0
	}
	return m
}

func (m Model) handleBufferDeleted(msg bufferDeletedMsg) Model {
	if msg.err != nil {
		m.errMsg = fmt.Sprintf("Failed to delete buffer: %v", msg.err)
		m.errFrames = 60
		return m
	}
	for i, s := range m.summaries {
		if s.ID == msg.id {
			m.summaries = append(m.summaries[:i], m.summaries[i+1:]...)
			if m.cursor >= len(m.summaries) && m.cursor > 0 {
				m.cursor--
			}
			if m.listOff > m.cursor {
				m.listOff = m.cursor
			}
			break
		}
	}
	m.deletingID = 0
	return m
}
