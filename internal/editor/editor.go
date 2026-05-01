package editor

import (
	"os"
	"os/exec"
	"strings"
)

// Resolve returns the editor command from preferred, $EDITOR, $VISUAL, or "vi".
func Resolve(preferred string) string {
	if preferred != "" {
		return preferred
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	return "vi"
}

// CreateTempFile writes content to a temp file and returns its path.
func CreateTempFile(content string) (string, error) {
	f, err := os.CreateTemp("", "tb-*.md")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// ReadFile reads the content of a file.
func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// BuildCmd creates an exec.Cmd for the editor on the given temp file.
// The editor string may include arguments (e.g. "code --wait").
func BuildCmd(editorStr, tmpPath string) *exec.Cmd {
	parts := strings.Fields(editorStr)
	if len(parts) == 0 {
		parts = []string{"vi"}
	}
	args := append(parts[1:], tmpPath)
	return exec.Command(parts[0], args...)
}
