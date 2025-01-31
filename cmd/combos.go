package cmd

import (
	"github.com/guettli/tff/pkg/tff"
	"github.com/spf13/cobra"
)

func init() {
	config := tff.CombosCmdConfig{}
	combosCmd := &cobra.Command{
		Use:   "combos",
		Short: "Conntect to one or several evdev devices and modify the events according to your configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			config.DevicePaths = args
			return tff.CombosMain(cmd.Context(), config)
		},
	}
	combosCmd.Flags().BoolVarP(&config.Debug, "debug", "d", false, "Print debug output")
	rootCmd.AddCommand(combosCmd)
}
