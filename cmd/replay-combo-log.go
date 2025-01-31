package cmd

import (
	"github.com/guettli/tff/pkg/tff"
	"github.com/spf13/cobra"
)

func init() {
	config := tff.CombosCmdConfig{}
	combosCmd := &cobra.Command{
		Use:   "reply-combo-log",
		Short: "Replay a combo log. Emit the events to the given log. This is useful for debugging.",
		RunE: func(cmd *cobra.Command, args []string) error {
			config.DevicePaths = args
			return tff.ReplayComboLogMain(cmd.Context(), config)
		},
	}
	combosCmd.Flags().BoolVarP(&config.Debug, "debug", "d", false, "Print debug output")
	rootCmd.AddCommand(combosCmd)
}
