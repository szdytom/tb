package cli

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/szdytom/tb/internal/config"
)

type rmFlags struct {
	permanent bool
}

func newRmCmd() *cobra.Command {
	var f rmFlags
	cmd := &cobra.Command{
		Use:   "rm <id>",
		Short: "Delete a buffer (moves to trash unless --permanent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(args[0], &f)
		},
	}
	cmd.Flags().BoolVarP(&f.permanent, "permanent", "P", false, "permanently delete immediately")
	return cmd
}

func runRm(idStr string, f *rmFlags) error {
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

	if f.permanent {
		err = client.PermanentlyDelete(id)
	} else {
		err = client.SoftDelete(id, cfg.TrashTTL)
	}
	if err != nil {
		printError(err.Error())
		return err
	}
	return nil
}
