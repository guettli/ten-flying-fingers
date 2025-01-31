package tff

import (
	"context"
	"fmt"

	"github.com/holoplot/go-evdev"
)

type CombosCmdConfig struct {
	Debug       bool
	DevicePaths []string
	ConfigFile  string
	Logfile     string
}

func ReplayComboLogMain(ctx context.Context, cmdconfig CombosCmdConfig) error {
	return replayComboLog(ctx, cmdconfig.ConfigFile, cmdconfig.Logfile)
}

func CombosMain(ctx context.Context, cmdconfig CombosCmdConfig) error {
	if len(cmdconfig.DevicePaths) == 0 {
		p, err := findDev()
		if err != nil {
			return err
		}
		fmt.Printf("Using device %q\n", p)
		cmdconfig.DevicePaths = []string{p}
	}
	combos, err := LoadYamlFile(cmdconfig.ConfigFile)
	if err != nil {
		return err
	}

	sourceDevs := make([]*evdev.InputDevice, 0, len(cmdconfig.DevicePaths))
	outDevs := make([]*evdev.InputDevice, 0, len(cmdconfig.DevicePaths))
	for i, path := range cmdconfig.DevicePaths {
		sourceDev, err := evdev.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open the source device: %q %w", path, err)
		}

		err = sourceDev.Grab()
		if err != nil {
			return err
		}
		outDev, err := evdev.CloneDevice(fmt.Sprintf("tff-clone-%d", i), sourceDev)
		if err != nil {
			return err
		}
		defer outDev.Close()
		sourceDevs = append(sourceDevs, sourceDev)
		outDevs = append(outDevs, outDev)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	errorChannel := make(chan error)
	for i := 0; i < len(cmdconfig.DevicePaths); i++ {
		go handleOneDevice(ctx, combos, sourceDevs[i], outDevs[i], cmdconfig.Debug, errorChannel)
	}
	err = <-errorChannel
	if err != nil {
		cancel(err)
	}
	for i := 0; i < len(cmdconfig.DevicePaths)-1; i++ {
		err := <-errorChannel
		fmt.Println(err)
	}
	return nil
}
