package config

import (
	"os"
	"path/filepath"
)

const appName = "tmpbuffer"

// DataDir returns $XDG_DATA_HOME/tmpbuffer (default ~/.local/share/tmpbuffer).
func DataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, appName)
	}

	return filepath.Join(os.Getenv("HOME"), ".local", "share", appName)
}

// ConfigDir returns $XDG_CONFIG_HOME/tmpbuffer (default ~/.config/tmpbuffer).
func ConfigDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, appName)
	}

	return filepath.Join(os.Getenv("HOME"), ".config", appName)
}

// SocketDir returns $XDG_STATE_HOME/tmpbuffer (default ~/.local/state/tmpbuffer).
func SocketDir() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, appName)
	}

	return filepath.Join(os.Getenv("HOME"), ".local", "state", appName)
}
