package tui

import "git.sr.ht/~rockorager/vaxis"

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

func classifyKey(ev vaxis.Key) keyAction {
	switch ev.String() {
	case "j", "Down":
		return keyDown
	case "k", "Up":
		return keyUp
	case "Page_Down", "space", "Ctrl+f":
		return keyPageDown
	case "Page_Up", "Ctrl+b":
		return keyPageUp
	case "g":
		return keyHome
	case "G":
		return keyEnd
	case "Enter":
		return keyEnter
	case "n":
		return keyNew
	case "d":
		return keyDelete
	case "?":
		return keyHelp
	case "y", "Y":
		return keyConfirm
	case "N", "Escape":
		return keyDeny
	case "/":
		return keySearch
	case "Ctrl+c":
		return keyQuit
	default:
		return keyNone
	}
}
