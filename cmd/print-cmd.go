package cmd

import (
	"github.com/guettli/tff/pkg/tff"
	"github.com/spf13/cobra"
)

func init() {
	combosCmd := &cobra.Command{
		Use:   "print [device]",
		Short: "Conntect to one evdev device and print the events",
		RunE: func(cmd *cobra.Command, args []string) error {
			device := ""
			if len(args) > 0 {
				device = args[0]
			}
			return tff.PrintMain(device)
		},
		Args:                  cobra.RangeArgs(0, 1),
		DisableFlagsInUseLine: true,
	}
	rootCmd.AddCommand(combosCmd)
}
