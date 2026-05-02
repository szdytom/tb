package tui

import (
	"strconv"
	"strings"
)

// executeCommand parses and dispatches a command entered via command mode (:).
func (a *App) executeCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	args := parts[1:]

	// Macro substitution: replace $ with the $ register value
	for i, arg := range args {
		if arg == "$" {
			args[i] = a.dollarReg
		}
	}

	switch cmd {
	case "q":
		a.execQuit()
	case "new":
		a.execNew()
	case "delete", "rm":
		a.execDelete(args)
	case "edit", "e":
		a.execEdit(args)
	case "help", "h":
		a.execHelp()
	default:
		a.setError("Unknown command: " + cmd)
	}
}

func (a *App) execQuit() {
	for _, tab := range a.editorTabs {
		tab.Close()
	}

	a.editorTabs = nil
	a.quitting = true
}

func (a *App) execNew() {
	a.createBufferAsync()
}

func (a *App) execDelete(args []string) {
	if len(args) > 0 {
		// :delete <id>
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			a.setError("Invalid buffer ID: " + args[0])

			return
		}

		a.deleteBufferAsync(id)

		return
	}

	if len(a.summaries) > 0 {
		a.deleteBufferAsync(a.summaries[a.cursor].ID)
	}
}

func (a *App) execEdit(args []string) {
	if len(args) > 0 {
		// :edit <id> — find buffer by ID
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			a.setError("Invalid buffer ID: " + args[0])

			return
		}

		for _, s := range a.summaries {
			if s.ID == id {
				a.startEditorAsync(id)

				return
			}
		}

		a.setError("Buffer not found: " + args[0])

		return
	}

	// :edit — open current buffer
	if len(a.summaries) > 0 {
		a.startEditorAsync(a.summaries[a.cursor].ID)
	}
}

func (a *App) execHelp() {
	a.curState = stateHelp
}
