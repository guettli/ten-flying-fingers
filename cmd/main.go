package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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

type Event = evdev.InputEvent

type KeyCode = evdev.EvCode

func timeSub(e1, e2 Event) time.Duration {
	return syscallTimevalToTime(e1.Time).Sub(syscallTimevalToTime(e2.Time))
}

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

			lines = append(lines, fmt.Sprintf("%s %s %+v %+v", d.Path(), name,
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

func usage() {
	fmt.Printf(`Create a new input device from an existing one
Usage:
  %s print [ /dev/input/... ]

      print events.
	  If no device was given, then the programm listens to all device and asks for a key press.

  %s csv [ /dev/input/... ]

     Write the events in CSV format.
	 If no device was given, then the programm listens to all device and asks for a key press.

  %s create-events-from-csv myfile.csv

     Create events from a csv file.

  Devices which look like a keyboard:
%s
`, os.Args[0], os.Args[0], os.Args[0], listDevices())
}

func findDev() (string, error) {
	dev_input := "/dev/input"
	entries, err := os.ReadDir(dev_input)
	if err != nil {
		return "", err
	}
	c := make(chan eventOfPath)
	foundDevices := 0
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
		foundDevices++
		defer func(dev *evdev.InputDevice, path string) {
			dev.Close()
		}(dev, path)
		go readEvents(dev, path, c)
	}
	if foundDevices == 0 {
		return "", fmt.Errorf("No device found")
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
		if !strings.HasPrefix(ev.CodeName(), "KEY_") {
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
	event *Event
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
	err := myMain()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func getDevicePathFromArgsSlice(args []string) (*evdev.InputDevice, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments")
	}
	path := ""
	if len(args) == 0 {
		p, err := findDev()
		if err != nil {
			return nil, err
		}
		fmt.Printf("Using device %q\n", p)
		path = p
	} else {
		path = args[0]
	}
	sourceDev, err := evdev.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open the source device: %w", err)
	}
	return sourceDev, nil
}

func myMain() error {
	defer os.Stdout.Close()
	if len(os.Args) < 2 {
		usage()
		return nil
	}

	cmd := os.Args[1]

	switch cmd {
	case "print":
		sourceDev, err := getDevicePathFromArgsSlice(os.Args[2:len(os.Args)])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		err = printEvents(sourceDev)
		if err != nil {
			fmt.Println(err.Error())
		}
		return nil
	case "csv":
		sourceDev, err := getDevicePathFromArgsSlice(os.Args[2:len(os.Args)])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		err = csv(sourceDev)
		if err != nil {
			fmt.Println(err.Error())
		}
		return nil
	case "mitm":
		sourceDev, err := getDevicePathFromArgsSlice(os.Args[2:len(os.Args)])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		return mitm(sourceDev)
	case "create-events-from-csv":
		if len(os.Args) != 3 {
			usage()
			return nil
		}
		err := createEventsFromCsv(os.Args[2])
		if err != nil {
			fmt.Println(err.Error())
		}
		return nil
	default:
		usage()
		return nil
	}
}

func createEventsFromCsv(csvPath string) error {
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", csvPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		ev, err := lineToEvent(line)
		if err != nil {
			return err
		}
		fmt.Println(ev.String())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading %q: %w", csvPath, err)
	}
	return nil
}

func lineToEvent(line string) (Event, error) {
	var ev Event
	parts := strings.Split(line, ";")
	if len(parts) != 5 {
		return ev, fmt.Errorf("failed to parse csv line: %s", line)
	}
	// 		// InputEvent describes an event that is generated by an InputDevice
	// type InputEvent struct {
	// 	Time  syscall.Timeval // time in seconds since epoch at which event occurred
	// 	Type  EvType          // event type - one of ecodes.EV_*
	// 	Code  EvCode          // event code related to the event type
	// 	Value int32           // event value related to the event type
	// }

	// type Timeval struct {
	// 	Sec  int64
	// 	Usec int64
	// }
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return ev, fmt.Errorf("failed to parse col 1 (sec) from line: %s. %w", line, err)
	}

	usec, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return ev, fmt.Errorf("failed to parse col 2 (usec) from line: %s. %w", line, err)
	}

	evType, ok := evdev.EVFromString[parts[2]]
	if !ok {
		return ev, fmt.Errorf("failed to parse col 3 (EvType) from line: %s. %s", line, parts[2])
	}

	code, ok := evdev.KEYFromString[parts[3]]
	if !ok {
		return ev, fmt.Errorf("failed to parse col 4 (Key) from line: %s. %s", line, parts[3])
	}
	var value int64
	switch parts[4] {
	case "up":
		value = UP
	case "down":
		value = DOWN
	case "repeat":
		value = REPEAT
	default:
		value, err = strconv.ParseInt(parts[4], 10, 32)
		if err != nil {
			return ev, fmt.Errorf("failed to parse col 5 (value) from line: %s. %w", line, err)
		}
	}
	return Event{
		Time:  syscall.Timeval{Sec: sec, Usec: usec},
		Type:  evType,
		Code:  code,
		Value: int32(value),
	}, nil
}

type Combo struct {
	Keys    []KeyCode
	OutKeys []KeyCode
}

func CopyCombos(combos []Combo) []Combo {
	newCombos := make([]Combo, 0, len(combos))
	for i := range combos {
		newCombos = append(newCombos, Combo{
			Keys:    slices.Clone(combos[i].Keys),
			OutKeys: slices.Clone(combos[i].OutKeys),
		})
	}
	return newCombos
}

// Man in the Middle.
func mitm(sourceDev *evdev.InputDevice) error {
	err := sourceDev.Grab()
	if err != nil {
		return err
	}
	outDev, err := evdev.CloneDevice("clone", sourceDev)
	if err != nil {
		return err
	}
	defer outDev.Close()
	allCombos := []Combo{
		{
			Keys:    []KeyCode{evdev.KEY_F, evdev.KEY_J},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	}
	return manInTheMiddle(sourceDev, outDev, allCombos)
}

type EventReader interface {
	ReadOne() (*Event, error)
}
type EventWriter interface {
	WriteOne(event *Event) error
}

func manInTheMiddle(er EventReader, ew EventWriter, allCombos []Combo) error {
	maxLength := 0
	for i := range allCombos {
		l := len(allCombos[i].Keys)
		if l > maxLength {
			maxLength = l
		}
	}
	if maxLength == 0 {
		return fmt.Errorf("No combo contains keys")
	}
	buffer := NewBuffer(maxLength, ew)
	combos := CopyCombos(allCombos)
	for {
		evP, err := er.ReadOne()
		if err != nil {
			if errors.Is(err, io.EOF) {
				buffer.FlushBuffer()
			}
			return err
		}
		if evP.Type != evdev.EV_KEY {
			err = ew.WriteOne(evP)
			if err != nil {
				return err
			}
		}
		switch evP.Value {
		case UP:
			combos, err = buffer.HandleUpChar(*evP, combos)
		case DOWN:
			combos, err = buffer.HandleDownChar(*evP, combos)
		default:
			return fmt.Errorf("Received %d. Expected UP or DOWN", evP.Value)
		}
		if err != nil {
			return err
		}
		if len(combos) == 0 {
			combos = CopyCombos(allCombos)
		}
	}
}

func NewBuffer(maxLength int, ew EventWriter) *Buffer {
	b := Buffer{
		outDev: ew,
	}
	b.buf = make([]Event, 0, maxLength)
	return &b
}

type Buffer struct {
	buf    []Event
	outDev EventWriter
}

func (b *Buffer) WriteEvent(ev Event) error {
	return b.outDev.WriteOne(&ev)
}

func (b *Buffer) ContainsKey(key KeyCode) bool {
	for i := range b.buf {
		if b.buf[i].Code == evdev.EvCode(key) {
			return true
		}
	}
	return false
}

func (b *Buffer) Len() int {
	return len(b.buf)
}

// The buffered events don't match a combo.
// Write out the buffered events and the current event.
func (b *Buffer) FlushBuffer() error {
	for _, bufEvent := range b.buf {
		if err := b.WriteEvent(bufEvent); err != nil {
			return err
		}
	}
	b.buf = nil
	return nil
}

func (b *Buffer) FlushBufferAndWriteEvent(ev Event) error {
	err := b.FlushBuffer()
	if err != nil {
		return err
	}
	return b.WriteEvent(ev)
}

func (b *Buffer) HandleUpChar(
	ev Event,
	combos []Combo,
) (
	combosOut []Combo,
	err error,
) {
	if b.Len() == 0 {
		// no Combo active. Write event, return
		return combos, b.WriteEvent(ev)
	}
	var downEvent *Event
	for i := range b.buf {
		if b.buf[i].Value == DOWN && b.buf[i].Code == ev.Code {
			downEvent = &b.buf[i]
			break
		}
	}
	if downEvent == nil {
		panic("no down?")
		// No corresponding downEvent. Write it out.
		return combos, b.WriteEvent(ev)
	}
	for _, combo := range combos {
		if len(combo.Keys) == 0 {
			lastEvent := b.buf[len(b.buf)-1]
			if timeSub(ev, lastEvent) < 50*time.Millisecond {
				panic("no short")
				// short overlap. This seems to be two characters
				// after each other, not a combo.
				// Write out all buffered events.
				return nil, b.FlushBufferAndWriteEvent(ev)
			}

			// All keys of this combo got pressed.
			err := b.WriteCombo(combo)
			if err != nil {
				return nil, err
			}
			b.buf = nil
			return nil, nil
		}
	}
	// no combo has matched. Write event to buffer
	panic("no match???")
	b.buf = append(b.buf, ev)
	return combos, nil
}

func (b *Buffer) WriteCombo(combo Combo) error {
	for i := range combo.OutKeys {
		if err := b.WriteEvent(Event{
			Time:  timeToSyscallTimeval(time.Now()),
			Type:  evdev.EV_KEY,
			Code:  evdev.EvCode(combo.OutKeys[i]),
			Value: DOWN,
		}); err != nil {
			return err
		}
		if err := b.WriteEvent(Event{
			Time:  timeToSyscallTimeval(time.Now()),
			Type:  evdev.EV_KEY,
			Code:  evdev.EvCode(combo.OutKeys[i]),
			Value: UP,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *Buffer) HandleDownChar(
	ev Event,
	combos []Combo,
) (
	combosOut []Combo,
	err error,
) {
	fmt.Printf("down %s\n", eventToCsvLine(ev))
	var openCombos []Combo
	// Does the current event match to a combo?
	for _, combo := range combos {
		for k := range combo.Keys {
			if combo.Keys[k] == ev.Code {
				// reduce active combos
				combo.Keys = slices.Delete(combo.Keys, k, k+1)
				openCombos = append(openCombos, combo)
				fmt.Printf("down-char matches combo, reduced key %s len: %d\n", eventToCsvLine(ev), len(combo.Keys))
				break
			}
		}
	}
	if len(openCombos) == 0 {
		panic("........")
		// No combo is matched.
		if b.Len() == 0 {
			// No combo was active. Write out event
			return combos, b.WriteEvent(ev)
		}
		// combo was not finished. Write out buffer and
		// reset combos to allCombos.
		return nil, b.FlushBufferAndWriteEvent(ev)
	}
	// At least one Combo is active.
	b.buf = append(b.buf, ev)
	return openCombos, nil
}

func csv(sourceDev *evdev.InputDevice) error {
	defer sourceDev.Close()
	targetName, err := sourceDev.Name()
	if err != nil {
		return err
	}
	fmt.Printf("#Reading %s %s\n", targetName, time.Now().String())
	for {
		ev, err := sourceDev.ReadOne()
		if err != nil {
			return err
		}
		if ev.Type == evdev.EV_SYN {
			continue
		}

		line := eventToCsvLine(*ev)
		fmt.Print(line)
	}
}

func printEvents(sourceDevice *evdev.InputDevice) error {
	defer sourceDevice.Close()
	sourceDevice.Grab()
	targetName, err := sourceDevice.Name()
	if err != nil {
		return err
	}
	fmt.Printf("Grabbing %s\n", targetName)
	fmt.Printf("Hold `x` for 1 second to exit.\n")
	prevEvent := Event{
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
	event *Event
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

func (s *Source) getOneEventOrTimeout(timeout time.Duration) (ev *Event, timedOut bool, err error) {
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

func eventKeyToString(ev *Event) (string, error) {
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

func csvToSlice(csvString string) ([]Event, error) {
	lines := strings.Split(csvString, "\n")
	s := make([]Event, 0, len(lines))
	for _, line := range lines {
		if line == "" || string(line[0]) == "#" {
			continue
		}
		ev, err := lineToEvent(line)
		if err != nil {
			return nil, err
		}
		s = append(s, ev)
	}
	return s, nil
}

func eventToCsvLine(ev Event) string {
	value := ""
	switch ev.Value {
	case DOWN:
		value = "down"
	case UP:
		value = "up"
	case REPEAT:
		value = "repeat"
	default:
		value = fmt.Sprint(ev.Value)
	}
	return fmt.Sprintf("%d;%d;%s;%s;%s\n", ev.Time.Sec, ev.Time.Usec,
		ev.TypeName(), ev.CodeName(),
		value)
}

func eventsToCsv(s []Event) string {
	csv := make([]string, 0, len(s))
	for _, ev := range s {
		csv = append(csv, eventToCsvLine(ev))
	}
	return strings.Join(csv, "")
}
