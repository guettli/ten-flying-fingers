package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

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
  %s print [ /dev/input/... | detect ] (use "detect" to interactively choose the device)

  Devices which look like a keyboard:
%s
`, os.Args[0], listDevices())
}

func findDev() (string, error) {
	dev_input := "/dev/input"
	entries, err := os.ReadDir(dev_input)
	if err != nil {
		return "", err
	}
	c := make(chan eventOfPath)
	for _, entry := range entries {
		if entry.Type()&os.ModeCharDevice == 0 {
			// not a character device file.
			continue
		}
		path := filepath.Join(dev_input, entry.Name())
		dev, err := evdev.Open(path)
		if err != nil {
			if strings.Contains(err.Error(), "inappropriate ioctl for device") {
				continue
			}
			fmt.Printf("failed to open %q: %s \n", path, err.Error())
			continue
		}
		defer func(dev *evdev.InputDevice, path string) {
			dev.Close()
		}(dev, path)
		go readEvents(dev, path, c)
	}
	fmt.Println("Please use the device you want to use, now. Capturing events ....")
	found := ""
	for {
		evOfPath := <-c
		ev := evOfPath.event
		if ev.Type != evdev.EV_KEY {
			continue
		}
		if ev.Value != UP {
			continue
		}
		found = evOfPath.path
		break
	}
	if found == "" {
		return "", fmt.Errorf("no device found which creates keyboard events")
	}
	return found, nil
}

type eventOfPath struct {
	path  string
	event *evdev.InputEvent
}

func readEvents(dev *evdev.InputDevice, path string, c chan eventOfPath) {
	for {
		ev, err := dev.ReadOne()
		if err != nil {
			return
		}
		c <- eventOfPath{path, ev}
	}
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
		path := ""
		if os.Args[2] == "detect" {
			p, err := findDev()
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			path = p
			fmt.Printf("Using device %q\n", path)
		} else {
			path = os.Args[2]
		}
		source, clone, err := cloneDevice(path)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		err = printEvents(source, clone)
		if err != nil {
			fmt.Println(err.Error())
		}
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

func printEvents(sourceDevice *evdev.InputDevice, clone *evdev.InputDevice) error {
	defer sourceDevice.Close()
	defer clone.Close()
	sourceDevice.Grab()
	targetName, err := sourceDevice.Name()
	if err != nil {
		return err
	}
	fmt.Printf("Grabbing %s\n", targetName)
	fmt.Printf("Hold `x` for 1 second to exit.\n")
	prevEvent := evdev.InputEvent{
		Time:  timeToSyscallTimeval(time.Now()),
		Type:  evdev.EV_KEY,
		Code:  evdev.KEY_SPACE,
		Value: UP,
	}
	source := Source{
		inputDevice:  sourceDevice,
		eventChannel: make(chan *ReadResult),
	}
	go source.readAndWriteForever()
	for {
		ev, timedOut, err := source.getOneEventOrTimeout(time.Duration(time.Second))
		if err != nil {
			return err
		}
		if timedOut {
			fmt.Println()
			duration := time.Since(syscallTimevalToTime(prevEvent.Time))
			if duration > time.Duration(10*time.Second) {
				fmt.Println("timeout")
				break
			}
			if duration > time.Second &&
				prevEvent.Code == evdev.KEY_X &&
				prevEvent.Value == DOWN {
				fmt.Println("exit")
				break
			}
			continue
		}
		if ev.Type == evdev.EV_SYN {
			continue
		}

		duration := time.Duration(ev.Time.Nano() - prevEvent.Time.Nano())
		// fmt.Printf("%v %v %v\n", ev.Time.Nano(), prevTime, duration.String())

		if duration > time.Second &&
			ev.Code == evdev.KEY_X &&
			ev.Value == UP {
			fmt.Println("exit")
			break
		}
		if err != nil {
			return err
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

		deltaCode := "" // if down/up keys overlap
		if ev.Value == UP &&
			prevEvent.Value == DOWN &&
			prevEvent.Code != ev.Code {
			deltaCode = fmt.Sprintf("(overlap %s->%s)", prevEvent.CodeName(), ev.CodeName())
		}

		fmt.Printf("%4dms  %s  %s\n", duration.Milliseconds(), s, deltaCode)
		if ev.Value == UP && ev.Code == evdev.KEY_SPACE {
			fmt.Println()
		}
		prevEvent = *ev
	}
	return nil
}

func timeToSyscallTimeval(t time.Time) syscall.Timeval {
	return syscall.Timeval{
		Sec:  int64(t.Unix()),              // Seconds since Unix epoch
		Usec: int64(t.Nanosecond() / 1000), // Nanoseconds to microseconds

	}
}

func syscallTimevalToTime(tv syscall.Timeval) time.Time {
	return time.Unix(tv.Sec, tv.Usec*1000)
}

type ReadResult struct {
	event *evdev.InputEvent
	err   error
}
type Source struct {
	inputDevice  *evdev.InputDevice
	eventChannel chan *ReadResult
}

func (s *Source) readAndWriteForever() {
	for {
		ev, err := s.inputDevice.ReadOne()
		s.eventChannel <- &ReadResult{ev, err}
	}
}

func (s *Source) getOneEventOrTimeout(timeout time.Duration) (ev *evdev.InputEvent, timedOut bool, err error) {
	select {
	case readResult := <-s.eventChannel:
		return readResult.event, false, readResult.err
	case <-time.After(timeout):
		return nil, true, nil
	}
}

var shortKeyNames = map[string]string{
	"space":      "␣",
	"leftshift":  "⇧ ",
	"rightshift": " ⇧",
}

func eventKeyToString(ev *evdev.InputEvent) (string, error) {
	if ev.Type != evdev.EV_KEY {
		return "", fmt.Errorf("need a EV_KEY event. Got %s", ev.String())
	}
	name := ev.CodeName()
	name = strings.TrimPrefix(name, "KEY_")
	name = strings.ToLower(name)
	short, ok := shortKeyNames[name]
	if ok {
		name = short
	}
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
