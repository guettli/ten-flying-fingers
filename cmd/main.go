package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/holoplot/go-evdev"
)

func listDevices() string {
	basePath := "/dev/input"

	files, err := os.ReadDir(basePath)
	if err != nil {
		return err.Error()
	}

	var lines []string
	foundOne := false
	for _, fileName := range files {
		if fileName.IsDir() {
			continue
		}
		full := fmt.Sprintf("%s/%s", basePath, fileName.Name())
		if d, err := evdev.OpenWithFlags(full, os.O_RDONLY); err == nil {
			foundOne = true
			name, _ := d.Name()

			// At least on my laptop many devices can emit EV_KEY.
			// So how to distuingish between a real keyboard and a device
			// like a power-button?
			// I found that EV_REP (repeated keys) are emitted only by keyboards.
			// Feel free to improve that.
			if !slices.Contains(d.CapableTypes(), evdev.EV_REP) {
				continue
			}

			lines = append(lines, fmt.Sprintf("%s: %s %+v %+v", d.Path(), name,
				Map(d.CapableTypes(), evdev.TypeName),
				Map(d.Properties(), evdev.PropName),
			))
			d.Close()
		}
	}
	if !foundOne {
		return "No single device was found. It is likely that you have no permission to access /dev/input/... (`sudo` might help)\n"
	}
	return strings.Join(lines, "\n")
}

func cloneDevice(devicePath string) (targetDev *evdev.InputDevice, cloneDev *evdev.InputDevice, err error) {
	targetDev, err = evdev.Open(devicePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open target device for cloning: %s", err.Error())
	}

	clonedDev, err := evdev.CloneDevice("my-device-clone", targetDev)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to clone device: %s", err.Error())
	}
	return targetDev, clonedDev, nil
}

func usage() {
	fmt.Printf(`Create a new input device from an existing one
Usage: 
  %s clone /dev/input/...

  Devices which look like a keyboard:
%s
`, os.Args[0], listDevices())
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "clone":
		if len(os.Args) < 3 {
			usage()
			return
		}
		target, clone, err := cloneDevice(os.Args[2])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		handle(target, clone)

	default:
		usage()
		return
	}
}

func handle(target *evdev.InputDevice, clone *evdev.InputDevice) error {
	defer target.Close()
	defer clone.Close()
	for {
		ev, err := target.ReadOne()
		if err != nil {
			return err
		}
		fmt.Printf("event %+v\n", ev)
		clone.WriteOne(ev)
	}
}

func Map[T any, S any](t []T, f func(T) S) []S {
	ret := make([]S, 0, len(t))
	for i := range t {
		ret = append(ret, f(t[i]))
	}
	return ret
}
