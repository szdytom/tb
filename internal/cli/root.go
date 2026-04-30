package cli

import (
	"github.com/spf13/cobra"
)

var jsonOutput bool

// newRootCmd creates the root cobra command for tb.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "tb",
		Short:         "tmpbuffer — a terminal-based text buffer manager",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	return cmd
}

// Execute builds the command tree and runs it with the given args.
// Returns an exit code (0 for success, 1 for error).
func Execute(args []string) int {
	root := newRootCmd()
	root.AddCommand(
		newAddCmd(),
		newListCmd(),
		newGetCmd(),
		newSearchCmd(),
		newEditCmd(),
		newRmCmd(),
		newPipeCmd(),
		newDaemonCmd(),
		newVersionCmd(),
	)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		printError(err.Error())
		return 1
	}
	return 0
}
