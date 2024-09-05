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

type KeyCode = evdev.EvCode // for exmaple KEY_A, KEY_B, ...

// timSub calculates the duration. The event "first" must be younger.
func timeSub(first, second Event) time.Duration {
	t1 := syscallTimevalToTime(first.Time)
	t2 := syscallTimevalToTime(second.Time)
	diff := t2.Sub(t1)
	if diff < 0 {
		panic(fmt.Sprintf("I am confused. timeSub should be a positive duration: diff=%v t1=%v t2=%v", diff, t1, t2))
	}
	return diff
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

  %s combos combos.yaml [ /dev/input/... ]

     Run combos defined in combos.yaml

  %s replay-combo-log combos.yaml combo.log

     Replay a combo log. If you got a panic while using the combos sub-command,
	 you can update the Go code and replay the log to see if the bug was fixed.
	 You must run the combos sub-command with the --debug flag to create the log.

  Devices which look like a keyboard:
%s
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], listDevices())
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
		return "", fmt.Errorf("No device found (try `sudo`, since root permissions are needed)")
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
	case "combos":
		var args []string
		debug := false
		for _, arg := range os.Args {
			if arg == "--debug" || arg == "-d" {
				debug = true
				continue
			}
			args = append(args, arg)
		}

		if len(args) != 3 && len(args) != 4 {
			fmt.Println("Not enough arguments")
			os.Exit(1)
		}
		sourceDev, err := getDevicePathFromArgsSlice(args[3:])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		err = combos(args[2], sourceDev, debug)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		return nil
	case "replay-combo-log":
		if len(os.Args) != 4 {
			fmt.Println("Not enough arguments")
			os.Exit(1)
		}
		err := replayComboLog(os.Args[2], os.Args[3])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
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
		ev, err := csvlineToEvent(line)
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

func csvlineToEvent(line string) (Event, error) {
	var ev Event
	parts := strings.Split(line, ";")
	if len(parts) != 5 {
		return ev, fmt.Errorf("failed to parse csv line: %s", line)
	}
	// InputEvent describes an event that is generated by an InputDevice
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
		return ev, fmt.Errorf("failed to parse col 3 (EvType) from line: %s. %q", line, parts[2])
	}

	var code evdev.EvCode
	if parts[3] == "SYN_REPORT" {
		code = evdev.SYN_REPORT
	} else {
		code, ok = evdev.KEYFromString[parts[3]]
		if !ok {
			return ev, fmt.Errorf("failed to parse col 4 (Key) from line: %s. %q", line, parts[3])
		}
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

func (c *Combo) matches(ev Event) bool {
	return slices.Contains(c.Keys, ev.Code)
}

func (c *Combo) String() string {
	keys := make([]string, 0, len(c.Keys))
	for _, k := range c.Keys {
		keys = append(keys, evdev.CodeName(evdev.EV_KEY, k))
	}
	out := make([]string, 0, len(c.OutKeys))
	for _, k := range c.OutKeys {
		out = append(out, evdev.CodeName(evdev.EV_KEY, k))
	}
	return fmt.Sprintf("Keys: %+v OutKeys: %+v", keys, out)
}

type partialCombo struct {
	combo        *Combo
	seenDownKeys []evdev.EvCode
	seenUpKeys   []evdev.EvCode

	// the downKeys of the combo have already been written.
	// This happens if you press and hold a combo.
	downKeysWritten bool

	time syscall.Timeval
}

func (pc *partialCombo) AllSeen() bool {
	if len(pc.combo.Keys) < len(pc.seenDownKeys) {
		panic(fmt.Sprintf("I am confused. More seenDownKeys than combo has keys %d %d", len(pc.combo.Keys), len(pc.seenDownKeys)))
	}
	if len(pc.combo.Keys) == len(pc.seenDownKeys) {
		return true
	}
	return false
}

func (pc *partialCombo) String() string {
	keys := make([]string, len(pc.combo.Keys))
	for i, key := range pc.combo.Keys {
		keys[i] = evdev.KEYToString[key]
	}
	outKeys := make([]string, len(pc.combo.OutKeys))
	for i, key := range pc.combo.OutKeys {
		outKeys[i] = evdev.KEYToString[key]
	}
	var seenDown []string
	for _, key := range pc.seenDownKeys {
		seenDown = append(seenDown, evdev.KEYToString[key])
	}
	var seenUp []string
	for _, key := range pc.seenUpKeys {
		seenUp = append(seenUp, evdev.KEYToString[key])
	}
	return fmt.Sprintf("[%s -> %s (seenDown %q seenUp %q)]",
		strings.Join(keys, " "),
		strings.Join(outKeys, " "),
		strings.Join(seenDown, " "),
		strings.Join(seenUp, " "),
	)
}

func partialCombosToString(partialCombos []*partialCombo) string {
	s := make([]string, len(partialCombos))
	for i, pc := range partialCombos {
		s[i] = pc.String()
	}
	return strings.Join(s, " - ")
}

type EventReader interface {
	ReadOne() (*Event, error)
}
type EventWriter interface {
	WriteOne(event *Event) error
}

func manInTheMiddle(er EventReader, ew EventWriter, allCombos []Combo, debug bool, fakeActiveTimer bool) error {
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
	state := NewState(maxLength, ew)
	state.fakeActiveTimer = fakeActiveTimer
	type eventAndErr struct {
		evP *Event
		err error
	}
	eventChannel := make(chan eventAndErr)
	go func() {
		for {
			evP, err := er.ReadOne()
			eventChannel <- eventAndErr{evP, err}
			if err != nil {
				return
			}
		}
	}()
	for {
		select {
		case eventErr, ok := <-eventChannel:
			if !ok {
				panic("I don't expect this channel to get closed.")
			}
			err := eventErr.err
			evP := eventErr.evP

			if err != nil {
				if errors.Is(err, io.EOF) {
					err = errors.Join(err, state.FlushBuffer("EOF"))
				}
				return err
			}
			if debug {
				fmt.Printf("|>>%s\n", eventToCsvLine(*evP))
			}
			if state.fakeActiveTimer && state.fakeActiverTimerNextTime.Before(syscallTimevalToTime(evP.Time)) {
				if err := state.AfterTimer(); err != nil {
					return err
				}
				state.fakeActiverTimerNextTime = maxTime
			}

			err = manInTheMiddleInnerLoop(evP, ew, state, allCombos, debug)
			if err != nil {
				return err
			}
		case <-state.activeTimer:
			if err := state.AfterTimer(); err != nil {
				return err
			}
		}
	}
}

func manInTheMiddleInnerLoop(evP *Event, ew EventWriter, state *State,
	allCombos []Combo, debug bool,
) error {
	var err error
	if evP.Type != evdev.EV_KEY {
		// fmt.Printf(" skipping %s\n", evP.String())
		err = ew.WriteOne(evP)
		if err != nil {
			return err
		}
		return nil
	}
	state.AssertValid()
	switch evP.Value {
	case UP:
		err = state.HandleUpChar(*evP, allCombos)
	case DOWN:
		err = state.HandleDownChar(*evP, allCombos)
	default:
		return fmt.Errorf("Received %d. Expected UP or DOWN", evP.Value)
	}
	if err != nil {
		return err
	}
	state.AssertValid()
	if debug {
		fmt.Printf(" partialCombos (%d) %s. Buffer %q\n",
			len(state.partialCombos),
			partialCombosToString(state.partialCombos), state.String())
	}
	return nil
}

func NewState(maxLength int, ew EventWriter) *State {
	s := State{
		outDev: ew,
	}
	s.buf = make([]Event, 0, maxLength)
	s.fakeActiverTimerNextTime = maxTime
	return &s
}

// Maximum possible time.Time value
var maxTime = time.Unix(1<<63-62135596801, 999999999)

type State struct {
	buf                      []Event
	outDev                   EventWriter
	partialCombos            []*partialCombo
	completedPartialCombo    *partialCombo    // PartialCombo where all keys have been pressed.
	activeTimer              <-chan time.Time // fires N milliseconds after the last key-down-event.
	fakeActiveTimer          bool             // In tests the activeTimer will be faked by reading the time of the next event.
	fakeActiverTimerNextTime time.Time        // The next the fakeActiveTimer event will be fired.
}

// AfterTimer gets called N milliseconds after the last key-down-event.
func (state *State) AfterTimer() error {
	if state.completedPartialCombo == nil {
		// No combos was completed. Flush buffer and clear partial combos.
		state.FlushBuffer(fmt.Sprintf("AfterTimer no completedPartialCombo %s|", state.String()))
		return nil
	}
	fmt.Printf("AfterTime has work to do state: %s\n", state.String())
	// All keys were pressed, but not all up-keys seen yet.
	// Fire the down-keys for this combo.
	if !state.completedPartialCombo.downKeysWritten {
		combo := state.completedPartialCombo.combo
		for i := range combo.OutKeys {
			if err := state.WriteEvent(Event{
				Time:  state.buf[len(state.buf)-1].Time,
				Type:  evdev.EV_KEY,
				Code:  evdev.EvCode(combo.OutKeys[i]),
				Value: DOWN,
			}, "AfterTimer()"); err != nil {
				return err
			}
		}
	}
	fmt.Printf("   AfterTime post loop\n")
	state.activeTimer = nil
	state.fakeActiverTimerNextTime = maxTime
	state.completedPartialCombo.downKeysWritten = true
	return nil
}

func (state *State) WriteEvent(ev Event, reason string) error {
	fmt.Printf("WriteEvent %s %s\n", reason, eventKeyToString(&ev))
	err := state.outDev.WriteOne(&ev)
	return errors.Join(err, state.outDev.WriteOne(&Event{
		Time: ev.Time,
		Type: evdev.EV_SYN,
		Code: evdev.SYN_REPORT,
	}))
}

func (state *State) ContainsKey(key KeyCode) bool {
	for i := range state.buf {
		if state.buf[i].Code == evdev.EvCode(key) {
			return true
		}
	}
	return false
}

func (state *State) Len() int {
	return len(state.buf)
}

func (state *State) String() string {
	var ret []string
	var prev *Event
	for i := range state.buf {
		ev := state.buf[i]
		if prev != nil {
			ret = append(ret, fmt.Sprintf("(%s)", timeSub(*prev, ev).String()))
		}
		ret = append(ret, eventKeyToString(&ev))
		prev = &ev
	}
	return strings.Join(ret, " ")
}

// The buffered events don't match a combo.
// Write out the buffered events and the current event.
// And set partialCombos to nil.
func (state *State) FlushBuffer(reason string) error {
	for _, bufEvent := range state.buf {
		if err := state.WriteEvent(bufEvent, "FlushBuffer "+reason); err != nil {
			return err
		}
	}
	state.buf = nil
	state.partialCombos = nil
	state.completedPartialCombo = nil
	state.activeTimer = nil
	state.fakeActiverTimerNextTime = maxTime
	return nil
}

func (state *State) FlushBufferAndWriteEvent(ev Event, reason string) error {
	err := state.FlushBuffer("FlushBufferAndWriteEvent flush buff " + reason)
	if err != nil {
		return err
	}
	return state.WriteEvent(ev, "FlushBufferAndWriteEvent write ev "+reason)
}

func (state *State) HandleUpChar(
	ev Event,
	allCombos []Combo,
) error {
	// fmt.Printf("up %s\n", eventToCsvLine(ev))

	if state.Len() == 0 {
		if len(state.partialCombos) != 0 {
			panic(fmt.Sprintf("I am confused. Buffer is empty, and there are partialCombos: %+v", state.partialCombos))
		}
		return state.FlushBufferAndWriteEvent(ev, "no Combo active")
	}

	// find corresponding down-event
	var downEvent *Event
	for i := range state.buf {
		if state.buf[i].Value == DOWN && state.buf[i].Code == ev.Code {
			downEvent = &state.buf[i]
			break
		}
	}
	if downEvent == nil {
		// No corresponding downEvent. Write it out.
		return state.FlushBufferAndWriteEvent(ev, "no corresponding downEvent")
	}

	if len(state.buf) == 1 && state.buf[0].Code == ev.Code && state.buf[0].Value == DOWN {
		return state.FlushBufferAndWriteEvent(ev,
			fmt.Sprintf("single key down/up of a key of a combo. (Ev: %s)",
				eventKeyToString(&ev)))
	}
	// Check if the up-key is part of a active partialCombo
	for i := range state.partialCombos {
		pc := state.partialCombos[i]
		if !pc.combo.matches(ev) {
			continue
		}

		pc.seenUpKeys = append(pc.seenUpKeys, ev.Code)
		if len(pc.seenUpKeys) == len(pc.combo.Keys) {
			// The final up-Key arrived. Time to execute combo?
			isOverlap := state.isOverlapOrFinishedCombo(pc)
			fmt.Printf("---HandleUp Final Key arrived. %+v isOverlap %v downKeysWritten %v\n", pc.time, isOverlap, pc.downKeysWritten)
			if isOverlap {
				return state.FlushBufferAndWriteEvent(ev, "isOverlap, not combo")
			}
			// All keys of this combo got pressed.
			if state.completedPartialCombo != nil && state.completedPartialCombo != pc {
				fmt.Printf("pre second combo, write completedPartialCombo %+v\n", state.completedPartialCombo)
				state.WriteCombo(*state.completedPartialCombo.combo,
					state.completedPartialCombo.time,
					state.completedPartialCombo.downKeysWritten)
				state.completedPartialCombo = nil
			}

			err := state.WriteCombo(*pc.combo, pc.time, pc.downKeysWritten)
			if err != nil {
				return err
			}
			return nil
		}
	}
	// no combo was finished. Write event to buffer
	state.buf = append(state.buf, ev)
	return nil
}

// Combo or overlap? Get timedelta between last down event, and first up event.
// short overlap. This seems to be two characters
// after each other, not a combo.
// Write out all buffered events.
// returns true, if it is an short overlap of keys, and not a combo.
func (state *State) isOverlapOrFinishedCombo(pc *partialCombo) bool {
	var firstUp *Event
	for i := range state.buf {
		ev := state.buf[i]
		if ev.Value == UP && slices.Contains(pc.combo.Keys, ev.Code) {
			firstUp = &state.buf[i]
			break
		}
	}
	if firstUp == nil {
		panic("I am confused, firstUp not found.")
	}
	var latestDown *Event
	for i := range state.buf {
		ev := &state.buf[len(state.buf)-1-i]
		if ev.Value == DOWN && slices.Contains(pc.combo.Keys, ev.Code) {
			latestDown = ev
			break
		}
	}
	if latestDown == nil {
		panic("I am confused, latestDown not found.")
	}

	state.AssertValid()
	fmt.Printf("is overlap delta ??? %v %v. %s\n", latestDown.String(), firstUp.String(), state.String())
	delta := timeSub(*latestDown, *firstUp)
	if delta < 0 {
		panic(fmt.Sprintf("I am confused. Negative time delta? delta: %v. firstUp %+v latestDown %+v | %s", delta,
			firstUp, latestDown, state.String()))
	}
	if delta < 50*time.Millisecond {
		fmt.Printf("is overlap delta: %v. %s\n", delta, state.String())
		return true
	}
	fmt.Printf("is no overlap delta: %v. %s\n", delta, state.String())
	return false
}

func (state *State) WriteCombo(combo Combo, time syscall.Timeval, downKeysWritten bool) error {
	fmt.Printf("WriteCombo %s downKeysWritten %v\n", combo.String(), downKeysWritten)
	for i := range combo.OutKeys {
		if !downKeysWritten {
			if err := state.WriteEvent(Event{
				Time:  time,
				Type:  evdev.EV_KEY,
				Code:  evdev.EvCode(combo.OutKeys[i]),
				Value: DOWN,
			}, "WriteCombo down"); err != nil {
				return err
			}
		}
		if err := state.WriteEvent(Event{
			Time:  time,
			Type:  evdev.EV_KEY,
			Code:  evdev.EvCode(combo.OutKeys[i]),
			Value: UP,
		}, "WriteCombo up"); err != nil {
			return err
		}
	}
	state.buf = nil
	state.partialCombos = nil
	state.completedPartialCombo = nil
	state.activeTimer = nil
	state.fakeActiverTimerNextTime = maxTime
	return nil
}

func (state *State) HandleDownChar(
	ev Event,
	allCombos []Combo,
) error {
	// fmt.Printf("down %s\n", eventToCsvLine(ev))

	timeoutAfterDownDuration := 150 * time.Millisecond
	if state.fakeActiveTimer {
		// For testing.
		state.fakeActiverTimerNextTime = syscallTimevalToTime(ev.Time).Add(timeoutAfterDownDuration)
	} else {
		state.activeTimer = time.After(timeoutAfterDownDuration)
	}

	// Filter the existing open partialCombos.
	// Skip partialCombos which don't match to current event.
	var newPartialCombos []*partialCombo
	for i := range state.partialCombos {
		pc := state.partialCombos[i]
		if pc.combo.matches(ev) {
			if !slices.Contains(pc.seenDownKeys, ev.Code) {
				pc.seenDownKeys = append(pc.seenDownKeys, ev.Code)
			}
			if pc.AllSeen() {
				if state.completedPartialCombo != nil {
					fmt.Printf(" ... pre panic: %s\n", state.String())
					panic("I am confused. Two combos are completed.")
				}
				state.completedPartialCombo = pc
			}
			newPartialCombos = append(newPartialCombos, pc)
		}
	}

	// Does this key start a new combo?
	for _, combo := range allCombos {
		if !combo.matches(ev) {
			continue
		}
		// Skip this combo, if it is already active
		skip := false
		for _, pc := range newPartialCombos {
			if slices.Equal(pc.combo.Keys, combo.Keys) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		pc := partialCombo{
			combo:        &combo,
			seenDownKeys: []evdev.EvCode{ev.Code},
			time:         ev.Time,
		}
		newPartialCombos = append(newPartialCombos, &pc)
	}
	state.partialCombos = newPartialCombos
	if len(state.partialCombos) == 0 {
		// No combo is matched.
		return state.FlushBufferAndWriteEvent(ev, "no combo matched")
	}
	// At least one Combo is active.
	state.buf = append(state.buf, ev)
	return nil
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
		var s string
		switch ev.Type {
		case evdev.EV_KEY:
			s = eventKeyToString(ev)
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

func eventKeyToString(ev *Event) string {
	if ev.Type != evdev.EV_KEY {
		return fmt.Sprintf("[err: need a EV_KEY event. Got %s]", ev.String())
	}
	name := ev.CodeName()
	name = strings.TrimPrefix(name, "KEY_")
	name = strings.ToLower(name)
	short, ok := shortKeyNames[name]
	if ok {
		name = short
	}
	if ev.Value > 2 {
		return fmt.Sprintf("ev.Value is unknown %d. %s", ev.Value, ev.String())
	}

	name = name + keyEventValueToString[ev.Value]
	return name
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
		line := strings.TrimSpace(line)
		if line == "" || string(line[0]) == "#" {
			continue
		}
		ev, err := csvlineToEvent(line)
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
		if ev.Type == evdev.EV_SYN {
			continue
		}
		csv = append(csv, eventToCsvLine(ev))
	}
	return strings.Join(csv, "")
}

type ComboLogEventReader struct {
	scanner *bufio.Scanner
}

func (c *ComboLogEventReader) ReadOne() (*Event, error) {
	for {
		if !c.scanner.Scan() {
			if err := c.scanner.Err(); err != nil {
				return nil, fmt.Errorf("error reading: %w", err)
			}
			return nil, io.EOF
		}
		line := c.scanner.Text()
		idx := strings.Index(line, "|>>")
		if idx == -1 {
			continue
		}
		ev, err := csvlineToEvent(line[idx+3:])
		return &ev, err
	}
}

func replayComboLog(comboYamlFile string, logFile string) error {
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

	return manInTheMiddle(&logReader, outDev, combos, true, false)
}

func combos(yamlFile string, dev *evdev.InputDevice, debug bool) error {
	combos, err := LoadYamlFile(yamlFile)
	if err != nil {
		return err
	}
	err = dev.Grab()
	if err != nil {
		return err
	}
	outDev, err := evdev.CloneDevice("clone", dev)
	if err != nil {
		return err
	}
	defer outDev.Close()
	return manInTheMiddle(dev, outDev, combos, debug, false)
}

func (state *State) AssertValid() {
	for _, pc := range state.partialCombos {
		found := false
		for _, ev := range state.buf {
			if slices.Contains(pc.combo.Keys, ev.Code) {
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf(`state.buf contains a event.Code which is not any in partialCombos.
%s`, state.String()))
		}
	}
}
