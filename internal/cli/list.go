package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/ipc"
)

type listFlags struct {
	filter string
	regex  string
	since  string
	until  string
	limit  int
}

func newListCmd() *cobra.Command {
	var f listFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active buffers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(&f)
		},
	}
	cmd.Flags().StringVarP(&f.filter, "filter", "f", "", "filter by keyword")
	cmd.Flags().StringVar(&f.regex, "regex", "", "filter by regex pattern")
	cmd.Flags().StringVarP(&f.since, "since", "s", "", "only buffers updated after RFC 3339 timestamp")
	cmd.Flags().StringVarP(&f.until, "until", "u", "", "only buffers updated before RFC 3339 timestamp")
	cmd.Flags().IntVarP(&f.limit, "limit", "n", 0, "maximum number of results")
	return cmd
}

func runList(f *listFlags) error {
	cfg := config.Default()
	client, err := NewClient(cfg)
	if err != nil {
		printError(err.Error())
		return err
	}
	defer client.Close()

	// When --regex is set, use the Search operation.
	if f.regex != "" {
		results, err := client.Search(f.regex, true)
		if err != nil {
			printError(err.Error())
			return err
		}
		bufs := make([]*buffer.Buffer, 0, len(results))
		for _, r := range results {
			bufs = append(bufs, r.Buffer)
		}
		if jsonOutput {
			printJSON(bufs)
		} else {
			printBufferTable(bufs)
		}
		return nil
	}

	payload := ipc.ListBuffersPayload{
		Keyword: f.filter,
		Limit:   f.limit,
		Since:   f.since,
		Until:   f.until,
	}
	bufs, err := client.ListBuffers(payload)
	if err != nil {
		printError(err.Error())
		return err
	}

	if jsonOutput {
		printJSON(bufs)
	} else {
		printBufferTable(bufs)
	}
	return nil
}

func printBufferTable(bufs []*buffer.Buffer) {
	if len(bufs) == 0 {
		fmt.Fprintln(os.Stderr, "No buffers")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tLABEL\tCONTENT\tUPDATED")
	for _, buf := range bufs {
		label := buf.Label
		if label == "" {
			label = "-"
		}
		preview := firstLine(buf.Content)
		if len(preview) > 48 {
			preview = preview[:48] + "..."
		}
		updated := buf.UpdatedAt.Format(time.RFC3339)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", buf.ID, truncate(label, 16), preview, updated)
	}
	w.Flush()
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
