package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultHasDefaults(t *testing.T) {
	ResetForTesting()

	cfg := Default()

	if cfg.TrashTTL != 86400 {
		t.Errorf("expected default TrashTTL 86400, got %d", cfg.TrashTTL)
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	t.Setenv("TB_PREVIEW_CMD", "")
	t.Setenv("TB_TIME_FORMAT", "")

	configDir := filepath.Join(dir, "tmpbuffer")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	content := `
editor = "nano"
preview_command = "bat --color=always"
time_format = "absolute"
trash_ttl_seconds = 3600

[editors]
md = "typora"
json = "code --wait"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	ResetForTesting()

	cfg := Default()

	if cfg.Editor != "nano" {
		t.Errorf("expected editor 'nano', got %q", cfg.Editor)
	}

	if cfg.PreviewCommand != "bat --color=always" {
		t.Errorf("expected preview_command 'bat --color=always', got %q", cfg.PreviewCommand)
	}

	if cfg.TimeFormat != TimeFormatAbsolute {
		t.Errorf("expected time_format absolute, got %d", cfg.TimeFormat)
	}

	if cfg.TrashTTL != 3600 {
		t.Errorf("expected trash_ttl 3600, got %d", cfg.TrashTTL)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("VISUAL", "")
	t.Setenv("TB_PREVIEW_CMD", "")
	t.Setenv("TB_TIME_FORMAT", "")

	configDir := filepath.Join(dir, "tmpbuffer")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	// File says "nano", env says "vim" → env wins.
	content := `editor = "nano"`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("EDITOR", "vim")

	ResetForTesting()

	cfg := Default()

	if cfg.Editor != "vim" {
		t.Errorf("expected env editor 'vim', got %q", cfg.Editor)
	}
}

func TestEnvVISUALOverridesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("EDITOR", "")
	t.Setenv("TB_PREVIEW_CMD", "")
	t.Setenv("TB_TIME_FORMAT", "")

	configDir := filepath.Join(dir, "tmpbuffer")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	content := `editor = "nano"`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("VISUAL", "helix")

	ResetForTesting()

	cfg := Default()

	if cfg.Editor != "helix" {
		t.Errorf("expected env VISUAL editor 'helix', got %q", cfg.Editor)
	}
}

func TestEnvOverridesDefault(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	t.Setenv("TB_PREVIEW_CMD", "bat --color=always")
	t.Setenv("TB_TIME_FORMAT", "absolute")

	ResetForTesting()

	cfg := Default()

	if cfg.PreviewCommand != "bat --color=always" {
		t.Errorf("expected preview_command 'bat --color=always', got %q", cfg.PreviewCommand)
	}

	if cfg.TimeFormat != TimeFormatAbsolute {
		t.Errorf("expected time_format absolute, got %d", cfg.TimeFormat)
	}
}

func TestMissingFileIsOK(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	ResetForTesting()

	cfg := Default()

	// Should not panic or error, all fields should be defaults.
	if cfg.TrashTTL != 86400 {
		t.Errorf("expected default TrashTTL 86400, got %d", cfg.TrashTTL)
	}
}

func TestCaching(t *testing.T) {
	ResetForTesting()

	cfg1 := Default()
	cfg2 := Default()

	// Two calls to Default() should return instances with identical values
	// (they share the same cached overlay).
	if cfg1.TrashTTL != cfg2.TrashTTL {
		t.Errorf("cached TrashTTL mismatch: %d vs %d", cfg1.TrashTTL, cfg2.TrashTTL)
	}

	if cfg1.Editor != cfg2.Editor {
		t.Errorf("cached Editor mismatch: %q vs %q", cfg1.Editor, cfg2.Editor)
	}
}

func TestResetForTesting(t *testing.T) {
	t.Setenv("EDITOR", "vim")
	t.Setenv("VISUAL", "")

	ResetForTesting()

	cfg := Default()

	if cfg.Editor != "vim" {
		t.Fatalf("expected 'vim', got %q", cfg.Editor)
	}

	// Reset and call again — verify Reset doesn't panic
	// and returns a valid config.
	ResetForTesting()

	cfg2 := Default()
	_ = cfg2
}

func TestCustomConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir) // should be ignored
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	t.Setenv("TB_PREVIEW_CMD", "")
	t.Setenv("TB_TIME_FORMAT", "")

	customPath := filepath.Join(dir, "myconfig.toml")

	content := `editor = "custom-editor"`
	if err := os.WriteFile(customPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	ResetForTesting()
	SetConfigFile(customPath)

	cfg := Default()

	if cfg.Editor != "custom-editor" {
		t.Errorf("expected editor 'custom-editor', got %q", cfg.Editor)
	}
}

func TestMalformedFileIsOK(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	configDir := filepath.Join(dir, "tmpbuffer")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Write garbage.
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("{{broken toml {{{"), 0600); err != nil {
		t.Fatal(err)
	}

	ResetForTesting()

	cfg := Default()

	if cfg.TrashTTL != 86400 {
		t.Errorf("expected default TrashTTL 86400 after malformed file, got %d", cfg.TrashTTL)
	}
}
