package cli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/szdytom/tb/internal/config"
)

type pipeFlags struct {
	command string
	new     bool
}

func newPipeCmd() *cobra.Command {
	var f pipeFlags
	cmd := &cobra.Command{
		Use:   "pipe <id> --command <cmd>",
		Short: "Pipe buffer content to a command and capture output",
		Long: `Pipe buffer content to a shell command. Captures stdout as the new content.

Examples:
  tb pipe 123 --command "jq ."
  tb pipe 123 --command "sort" --new`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipe(args[0], &f)
		},
	}
	cmd.Flags().StringVar(&f.command, "command", "", "shell command to run (required)")
	cmd.MarkFlagRequired("command")
	cmd.Flags().BoolVarP(&f.new, "new", "n", false, "create a new buffer with the output")
	return cmd
}

func runPipe(idStr string, f *pipeFlags) error {
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
		printError(err.Error())
		return err
	}

	cmd := exec.Command("sh", "-c", f.command)
	cmd.Stdin = bytes.NewReader([]byte(buf.Content))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("command failed: %s", err.Error())
		if stderr.Len() > 0 {
			errMsg += "\n" + stderr.String()
		}
		printError(errMsg)
		return err
	}

	output := stdout.String()

	if f.new {
		newID, err := client.CreateBuffer(output, buf.Label+" (piped)", nil)
		if err != nil {
			printError(err.Error())
			return err
		}
		if !jsonOutput {
			fmt.Println(newID)
		} else {
			printJSON(map[string]int64{"id": newID})
		}
	} else {
		if err := client.UpdateContent(id, output); err != nil {
			printError(err.Error())
			return err
		}
	}

	return nil
}
