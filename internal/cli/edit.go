package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/szdytom/tb/internal/config"
)

type editFlags struct {
	editor string
}

func newEditCmd() *cobra.Command {
	var f editFlags

	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Open a buffer in the external editor",
		Long: `Open a buffer in $EDITOR (or --editor). Saves changes when the editor exits.

Examples:
  tb edit 123
  tb edit 123 --editor vim
  EDITOR=code --wait tb edit 123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEdit(args[0], &f)
		},
	}
	cmd.Flags().StringVarP(&f.editor, "editor", "e", "", "editor command (overrides $EDITOR)")

	return cmd
}

func runEdit(idStr string, f *editFlags) error {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		printError("invalid buffer id: " + idStr)

		return err
	}

	cfg := config.Default()

	client, err := NewClient(cfg)
	if err != nil {
		printError(err.Error())

		return err
	}
	defer client.Close()

	buf, err := client.GetBuffer(id)
	if err != nil {
		printError(fmt.Sprintf("buffer %d: %s", id, err.Error()))

		return err
	}

	tmpFile, err := os.CreateTemp("", "tb-*.md")
	if err != nil {
		printError("create temp file: " + err.Error())

		return err
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(buf.Content); err != nil {
		_ = tmpFile.Close()

		printError("write temp file: " + err.Error())

		return err
	}

	_ = tmpFile.Close()

	editorCmd := f.editor
	if editorCmd == "" {
		editorCmd = cfg.Editor
	}

	if editorCmd == "" {
		editorCmd = "vi"
	}

	return editWithEditor(client, id, tmpPath, buf.Content, editorCmd)
}

func editWithEditor(client *Client, id int64, tmpPath, origContent, editorCmd string) error {
	cmd := exec.Command(editorCmd, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "Editor exited with code %d. Keep changes? [y/N] ", exitErr.ExitCode())

			var response string

			_, _ = fmt.Scanln(&response)

			if response != "y" && response != "Y" {
				fmt.Fprintln(os.Stderr, "Changes discarded")

				return nil
			}
		} else {
			return fmt.Errorf("run editor: %w", err)
		}
	}

	newContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("read temp file: %w", err)
	}

	newStr := string(newContent)
	if newStr != origContent {
		return client.UpdateContent(id, newStr)
	}

	return nil
}
