package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the current version. Set via -ldflags at build time.
var Version = "1.0.0"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("tb version", Version)

			return nil
		},
	}
}
