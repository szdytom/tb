package config

import (
	"os"
	"path/filepath"
)

const (
	TimeFormatRelative = iota
	TimeFormatAbsolute
)

// Config holds all configuration for tmpbuffer.
type Config struct {
	DataDir        string
	ConfigDir      string
	SocketDir      string
	DBPath         string
	SocketPath     string
	Editor         string
	PreviewCommand string
	TimeFormat     int
}

func timeFormatFromString(s string) int {
	switch s {
	case "relative":
		return TimeFormatRelative
	case "absolute":
		return TimeFormatAbsolute
	default:
		return TimeFormatRelative
	}
}

// Default returns a Config populated from XDG conventions and environment.
// TODO: cache this since it's used in multiple places and doesn't change at runtime.
// TODO: load additional config from a file
func Default() *Config {
	return &Config{
		DataDir:        DataDir(),
		ConfigDir:      ConfigDir(),
		SocketDir:      SocketDir(),
		DBPath:         filepath.Join(DataDir(), "tmpbuffer.db"),
		SocketPath:     filepath.Join(SocketDir(), "tmpbuffer.sock"),
		Editor:         os.Getenv("EDITOR"),
		PreviewCommand: os.Getenv("TB_PREVIEW_CMD"),
		TimeFormat:     timeFormatFromString(os.Getenv("TB_TIME_FORMAT")),
	}
}

// EnsureDirs creates all required directories with restrictive permissions.
func (c *Config) EnsureDirs() error {
	for _, d := range []string{c.DataDir, c.ConfigDir, c.SocketDir} {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}
	return nil
}

// PidFilePath returns the path to the daemon PID file.
func (c *Config) PidFilePath() string {
	return filepath.Join(c.SocketDir, "tmpbuffer.pid")
}
