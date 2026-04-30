package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// printJSON writes v as indented JSON to stdout.
func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// printError prints an error message to stderr.
func printError(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
}

// isStdinTerminal returns true if stdin is a terminal (not a pipe).
func isStdinTerminal() bool {
	fi, _ := os.Stdin.Stat()
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// firstLine returns the first line of a string, or the whole string if
// there is only one line.
func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}
