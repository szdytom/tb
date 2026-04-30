package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/szdytom/tb/internal/config"
)

type addFlags struct {
	text  string
	label string
	tags  []string
}

func newAddCmd() *cobra.Command {
	var f addFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new buffer",
		Long: `Create a new buffer from stdin, --text, or empty.

Examples:
  echo "hello" | tb add
  tb add --text "hello" --label greeting
  tb add`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(&f)
		},
	}
	cmd.Flags().StringVarP(&f.text, "text", "T", "", "buffer content as text argument")
	cmd.Flags().StringVarP(&f.label, "label", "l", "", "human-readable label")
	cmd.Flags().StringSliceVarP(&f.tags, "tag", "t", nil, "tags (comma-separated)")
	return cmd
}

func runAdd(f *addFlags) error {
	var content string
	switch {
	case f.text != "":
		content = f.text
	case !isStdinTerminal():
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			printError("read stdin: " + err.Error())
			return err
		}
		content = string(data)
	}

	cfg := config.Default()
	client, err := NewClient(cfg)
	if err != nil {
		printError(err.Error())
		return err
	}
	defer client.Close()

	id, err := client.CreateBuffer(content, f.label, f.tags)
	if err != nil {
		printError(err.Error())
		return err
	}

	if jsonOutput {
		printJSON(map[string]int64{"id": id})
	} else {
		fmt.Println(id)
	}
	return nil
}
