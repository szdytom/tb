package config

import (
	"os"
	"path/filepath"
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
}

// Default returns a Config populated from XDG conventions and environment.
func Default() *Config {
	return &Config{
		DataDir:    DataDir(),
		ConfigDir:  ConfigDir(),
		SocketDir:  SocketDir(),
		DBPath:     filepath.Join(DataDir(), "tmpbuffer.db"),
		SocketPath: filepath.Join(SocketDir(), "tmpbuffer.sock"),
		Editor:         os.Getenv("EDITOR"),
		PreviewCommand: os.Getenv("TB_PREVIEW_CMD"),
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
