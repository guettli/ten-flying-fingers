package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/holoplot/go-evdev"
)

const (
	UP     = 0
	DOWN   = 1
	REPEAT = 2
)

var keyEventValueToString = []string{"/", "_", "="}

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

func cloneDevice(devicePath string) (sourceDev *evdev.InputDevice, cloneDev *evdev.InputDevice, err error) {
	sourceDev, err = evdev.Open(devicePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the source device for cloning: %s", err.Error())
	}

	clonedDev, err := evdev.CloneDevice("my-device-clone", sourceDev)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to clone device: %s", err.Error())
	}
	return sourceDev, clonedDev, nil
}

func usage() {
	fmt.Printf(`Create a new input device from an existing one
Usage: 
  %s print /dev/input/...

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
	case "print":
		if len(os.Args) < 3 {
			usage()
			return
		}
		source, clone, err := cloneDevice(os.Args[2])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		err = printEvents(source, clone)
		fmt.Println(err.Error())
		return
	case "record":
		if len(os.Args) < 3 {
			usage()
			return
		}
		err := record(os.Args[2])
		if err != nil {
			fmt.Println(err.Error())
		}
		return
	default:
		usage()
		return
	}
}

func record(targetPath string) error {
	source, clone, err := cloneDevice(targetPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer source.Close()
	defer clone.Close()
	source.Grab()
	targetName, err := source.Name()
	if err != nil {
		return err
	}
	fmt.Printf("Grabbing %s\n", targetName)
	for {
		ev, err := source.ReadOne()
		if err != nil {
			return err
		}
		if ev.Code == evdev.KEY_A {
			fmt.Println("I don't like that key")
			continue
		}
		fmt.Printf("event %+v\n", ev)
		clone.WriteOne(ev)
	}
}

func printEvents(source *evdev.InputDevice, clone *evdev.InputDevice) error {
	defer source.Close()
	defer clone.Close()
	source.Grab()
	targetName, err := source.Name()
	if err != nil {
		return err
	}
	fmt.Printf("Grabbing %s\n", targetName)
	for {
		ev, err := source.ReadOne()
		if err != nil {
			return err
		}
		if ev.Type == evdev.EV_SYN {
			continue
		}
		var s string
		switch ev.Type {
		case evdev.EV_KEY:
			s, err = eventKeyToString(ev)
			if err != nil {
				return err
			}
		default:
			s = ev.String()
		}
		fmt.Println(s)
	}
}

func eventKeyToString(ev *evdev.InputEvent) (string, error) {
	if ev.Type != evdev.EV_KEY {
		return "", fmt.Errorf("need a EV_KEY event. Got %s", ev.String())
	}
	name := ev.CodeName()
	name = strings.TrimPrefix(name, "KEY_")
	name = strings.ToLower(name)
	if ev.Value > 2 {
		return "", fmt.Errorf("ev.Value is unknown %d. %s", ev.Value, ev.String())
	}
	name = name + keyEventValueToString[ev.Value]
	return name, nil
}

func Map[T any, S any](t []T, f func(T) S) []S {
	ret := make([]S, 0, len(t))
	for i := range t {
		ret = append(ret, f(t[i]))
	}
	return ret
}
