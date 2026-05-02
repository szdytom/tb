package cli

import (
	"github.com/spf13/cobra"
	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/tui"
)

var (
	jsonOutput bool
	configFile string
)

// newRootCmd creates the root cobra command for tb.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "tb",
		Short:         "tmpbuffer — a terminal-based text buffer manager",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if configFile != "" {
				config.SetConfigFile(configFile)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to config file")
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
		newTuiCmd(),
	)
	root.SetArgs(args)

	if err := root.Execute(); err != nil {
		printError(err.Error())

		return 1
	}

	return 0
}

func runTUI() error {
	cfg := config.Default()

	client, err := NewClient(cfg)
	if err != nil {
		printError(err.Error())

		return err
	}
	defer client.Close()

	return tui.New(client, cfg.PreviewCommand, cfg.Editor, cfg.TrashTTL).Run()
}

func newTuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}
}
