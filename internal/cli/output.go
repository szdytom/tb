package cli

import (
	"encoding/json"
	"fmt"
	"os"
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
