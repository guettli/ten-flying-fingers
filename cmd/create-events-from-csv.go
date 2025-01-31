package cmd

import (
	"fmt"

	"github.com/guettli/tff/pkg/tff"
	"github.com/spf13/cobra"
)

func init() {
	combosCmd := &cobra.Command{
		Use:   "create-events-from-csv events.csv",
		Short: "Read events from csv file, and emit the events. This does not rewrite the events like 'replay-combo-log' does.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) > 0 {
				path = args[0]
			}
			err := tff.CreateEventsFromCsv(path)
			if err != nil {
				fmt.Println(err.Error())
			}
			return nil
		},
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
	}
	rootCmd.AddCommand(combosCmd)
}
