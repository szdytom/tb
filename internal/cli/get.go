package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/szdytom/tb/internal/config"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Retrieve a buffer's content by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(args[0])
		},
	}
}

func runGet(idStr string) error {
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

	if jsonOutput {
		printJSON(buf)
	} else {
		fmt.Print(buf.Content)
		if buf.Content != "" && buf.Content[len(buf.Content)-1] != '\n' {
			fmt.Println()
		}
	}
	return nil
}
