package tui

import tea "github.com/charmbracelet/bubbletea"

type keyAction int

const (
	keyNone keyAction = iota
	keyUp
	keyDown
	keyPageUp
	keyPageDown
	keyHome
	keyEnd
	keyEnter
	keyNew
	keyDelete
	keyQuit
	keyHelp
	keyConfirm
	keyDeny
	keySearch
)

func classifyKey(msg tea.KeyMsg) keyAction {
	switch msg.String() {
	case "j", "down":
		return keyDown
	case "k", "up":
		return keyUp
	case "pgdown", " ", "ctrl+f":
		return keyPageDown
	case "pgup", "ctrl+b":
		return keyPageUp
	case "g":
		return keyHome
	case "G":
		return keyEnd
	case "enter":
		return keyEnter
	case "n":
		return keyNew
	case "d":
		return keyDelete
	case "q":
		return keyQuit
	case "?":
		return keyHelp
	case "y", "Y":
		return keyConfirm
	case "N", "esc":
		return keyDeny
	case "/":
		return keySearch
	case "ctrl+c":
		return keyQuit
	default:
		return keyNone
	}
}
