package cmd

import (
	"fmt"
	"os"

	"github.com/guettli/tff/pkg/tff"
	"github.com/spf13/cobra"
)

func init() {
	combosCmd := &cobra.Command{
		Use:   "csv [device]",
		Short: "Conntect to one evdev device and print the events in csv format. Needs root permissions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) > 0 {
				path = args[0]
			}
			sourceDev, err := tff.GetDeviceFromPath(path)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			err = tff.Csv(sourceDev)
			if err != nil {
				fmt.Println(err.Error())
			}
			return nil
		},
		Args:                  cobra.RangeArgs(0, 1),
		DisableFlagsInUseLine: true,
	}
	rootCmd.AddCommand(combosCmd)
}
