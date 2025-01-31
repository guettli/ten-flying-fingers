package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tff",
	Short: "tff is tool to modify Linux evdev keyboard events. You can build custom shortcuts (combos) to get 'ten flying fingers'.",
	Long:  `tff (ten flying fingers). https://github.com/guettli/tff`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
