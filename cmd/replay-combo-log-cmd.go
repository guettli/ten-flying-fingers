package cmd

import (
	"github.com/guettli/tff/pkg/tff"
	"github.com/spf13/cobra"
)

func init() {
	combosCmd := &cobra.Command{
		Use:   "reply-combo-log combos.yaml combos.log",
		Short: "Replay a combo log. Emit the events from the given log. This is useful for debugging.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tff.ReplayComboLogMain(cmd.Context(), args[0], args[1])
		},
		Args:                  cobra.ExactArgs(2),
		DisableFlagsInUseLine: true,
	}
	rootCmd.AddCommand(combosCmd)
}
