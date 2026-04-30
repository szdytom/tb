package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/szdytom/tb/internal/config"
)

type searchFlags struct {
	regex bool
}

func newSearchCmd() *cobra.Command {
	var f searchFlags
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search across all buffers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(args[0], &f)
		},
	}
	cmd.Flags().BoolVarP(&f.regex, "regex", "r", false, "treat query as a regular expression")
	return cmd
}

func runSearch(query string, f *searchFlags) error {
	cfg := config.Default()
	client, err := NewClient(cfg)
	if err != nil {
		printError(err.Error())
		return err
	}
	defer client.Close()

	results, err := client.Search(query, f.regex)
	if err != nil {
		printError(err.Error())
		return err
	}

	if jsonOutput {
		printJSON(results)
		return nil
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No results")
		return nil
	}

	for _, r := range results {
		label := r.Buffer.Label
		if label == "" {
			label = "(no label)"
		}
		fmt.Fprintf(os.Stdout, "%d\t%s\n%s\n\n", r.Buffer.ID, label, r.Snippet)
	}
	return nil
}
