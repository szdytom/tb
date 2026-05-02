package config

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
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
	TrashTTL       int
}

// tomlConfig mirrors the config.toml file structure.
type tomlConfig struct {
	Editor         string `toml:"editor"`
	PreviewCommand string `toml:"preview_command"`
	TimeFormat     string `toml:"time_format"`
	TrashTTL       int    `toml:"trash_ttl_seconds"`
}

// configOverlay holds the subset of config loaded once from file + env.
// XDG paths are NOT cached so tests that set XDG_* env vars work correctly.
type configOverlay struct {
	Editor         string
	PreviewCommand string
	TimeFormat     int
	TrashTTL       int
}

var (
	overlayOnce      sync.Once
	overlay          *configOverlay
	customConfigFile string
)

// SetConfigFile sets a custom config file path, overriding the default
// $XDG_CONFIG_HOME/tmpbuffer/config.toml. Must be called before the first
// call to Default().
func SetConfigFile(path string) {
	customConfigFile = path
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

// Default returns a Config populated from XDG conventions, a TOML config file
// (if present), and environment variables. Environment variables override file
// values. The file+env overlay is cached after the first call.
func Default() *Config {
	overlayOnce.Do(func() {
		o := &configOverlay{
			TrashTTL: 86400,
		}
		loadFile(o)
		applyEnvOverrides(o)
		overlay = o
	})

	return &Config{
		DataDir:        DataDir(),
		ConfigDir:      ConfigDir(),
		SocketDir:      SocketDir(),
		DBPath:         filepath.Join(DataDir(), "tmpbuffer.db"),
		SocketPath:     filepath.Join(SocketDir(), "tmpbuffer.sock"),
		Editor:         overlay.Editor,
		PreviewCommand: overlay.PreviewCommand,
		TimeFormat:     overlay.TimeFormat,
		TrashTTL:       overlay.TrashTTL,
	}
}

// loadFile reads config.toml from the config directory and merges non-zero
// values into the overlay. Missing or malformed files are silently ignored.
func loadFile(o *configOverlay) {
	path := customConfigFile
	if path == "" {
		path = filepath.Join(ConfigDir(), "config.toml")
	}

	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	var tc tomlConfig
	if _, err := toml.NewDecoder(f).Decode(&tc); err != nil {
		return
	}

	if tc.Editor != "" {
		o.Editor = tc.Editor
	}

	if tc.PreviewCommand != "" {
		o.PreviewCommand = tc.PreviewCommand
	}

	if tc.TimeFormat != "" {
		o.TimeFormat = timeFormatFromString(tc.TimeFormat)
	}

	if tc.TrashTTL > 0 {
		o.TrashTTL = tc.TrashTTL
	}
}

// applyEnvOverrides applies environment variables on top of file-loaded values,
// so env always wins.
func applyEnvOverrides(o *configOverlay) {
	if e := os.Getenv("VISUAL"); e != "" {
		o.Editor = e
	}

	if e := os.Getenv("EDITOR"); e != "" {
		o.Editor = e
	}

	if e := os.Getenv("TB_PREVIEW_CMD"); e != "" {
		o.PreviewCommand = e
	}

	if e := os.Getenv("TB_TIME_FORMAT"); e != "" {
		o.TimeFormat = timeFormatFromString(e)
	}
}

// ResetForTesting resets the cached config overlay. Only used in tests.
func ResetForTesting() {
	overlay = nil
	overlayOnce = sync.Once{}
	customConfigFile = ""
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

// GetCustomConfigFile returns the custom config file path set via SetConfigFile, or empty.
func GetCustomConfigFile() string {
	return customConfigFile
}

// PidFilePath returns the path to the daemon PID file.
func (c *Config) PidFilePath() string {
	return filepath.Join(c.SocketDir, "tmpbuffer.pid")
}
