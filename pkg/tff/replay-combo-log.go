package tff

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/holoplot/go-evdev"
)

func ReplayComboLogMain(ctx context.Context, comboYamlFile string, logFile string) error {
	outDev, err := evdev.CreateDevice("replay", evdev.InputID{
		BusType: 0x03,
		Vendor:  0x4711,
		Product: 0x0816,
		Version: 1,
	}, nil)
	if err != nil {
		return err
	}
	defer outDev.Close()
	combos, err := LoadYamlFile(comboYamlFile)
	if err != nil {
		return fmt.Errorf("failed to load %q: %w", comboYamlFile, err)
	}
	file, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", logFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	logReader := ComboLogEventReader{scanner: scanner}

	return manInTheMiddle(ctx, &logReader, outDev, combos, false)
}
