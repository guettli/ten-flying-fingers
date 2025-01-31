package cmd

import (
	"errors"

	"github.com/guettli/tff/pkg/tff"
	"github.com/spf13/cobra"
)

func init() {
	config := tff.CombosCmdConfig{}
	combosCmd := &cobra.Command{
		Use:   "combos [flags] combos.yaml [device1 [device2 ...]]",
		Short: "Conntect to one or several evdev devices and modify the events according to your configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			config.ConfigFile = args[0]
			config.DevicePaths = args[1:]
			return tff.CombosMain(cmd.Context(), config)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("You need to provide at least one argument: combos.yaml")
			}
			return nil
		},
	}
	combosCmd.Flags().BoolVarP(&config.Debug, "debug", "d", false, "Print debug output")
	rootCmd.AddCommand(combosCmd)
}
