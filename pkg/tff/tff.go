package tff

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
func timeSub(first, second syscall.Timeval) time.Duration {
	t1 := syscallTimevalToTime(first)
	t2 := syscallTimevalToTime(second)
	diff := t2.Sub(t1)
	if diff < 0 {
		panic(fmt.Sprintf("I am confused. timeSub should be a positive duration: diff=%v t1=%v t2=%v", diff, t1, t2))
	}
	return diff
}

var (
	eventValueToShortString = []string{"/", "_", "="}
	eventValueToString      = map[int32]string{
		UP:     "up",
		DOWN:   "down",
		REPEAT: "repeat",
	}
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

  %s combos [--debug] combos.yaml [ /dev/input/... ]

     Run combos defined in combos.yaml

  %s replay-combo-log combos.yaml combo.log

     Replay a combo log. If you got a panic while using the combos sub-command,
	 you can update the Go code and replay the log to see if the bug was fixed.
	 You must run the 'combos' sub-command with the --debug flag to create the log.

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

func MyMain() error {
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
			return fmt.Errorf("failed to create event from csv: %w", err)
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

	// EV_KEY, EV_SYN, EV_MSC, ...
	evType, ok := evdev.EVFromString[parts[2]]
	if !ok {
		return ev, fmt.Errorf("failed to parse col 3 (EvType) from line: %s. %q", line, parts[2])
	}

	var code evdev.EvCode
	switch parts[3] {
	case "SYN_REPORT":
		code = evdev.SYN_REPORT
	case "MSC_SCAN":
		code = evdev.MSC_SCAN
	default:
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
	return fmt.Sprintf("%+v -> %+v", strings.Join(keys, " "), strings.Join(out, " "))
}

func keyToString(key KeyCode) string {
	return strings.TrimPrefix(evdev.KEYToString[key], "KEY_")
}

func SliceOfKeysToString(keys []KeyCode) string {
	s := make([]string, 0, len(keys))
	for _, key := range keys {
		s = append(s, keyToString(key))
	}
	return "[" + strings.Join(s, " ") + "]"
}

type EventReader interface {
	ReadOne() (*Event, error)
}
type EventWriter interface {
	WriteOne(event *Event) error
}

func manInTheMiddle(er EventReader, ew EventWriter, allCombos []*Combo, debug bool, fakeActiveTimer bool) error {
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
	state := NewState(maxLength, ew, allCombos)
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
			if evP.Code == evdev.KEY_RFKILL {
				// This is used for unit-tests.
				// It makes the endless loop stop without the final FlushBuffer.
				return io.EOF
			}
			if state.fakeActiveTimer && state.fakeActiverTimerNextTime.Before(syscallTimevalToTime(evP.Time)) {
				if err := state.fakeAfterTimerFunc(state.fakeActiverTimerNextTime); err != nil {
					return err
				}
				state.fakeActiverTimerNextTime = maxTime
			}

			if debug {
				fmt.Printf("\n|>>%s", eventToCsvLine(*evP))
			}

			err = manInTheMiddleInnerLoop(evP, ew, state)
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

func manInTheMiddleInnerLoop(evP *Event, ew EventWriter, state *State) error {
	var err error
	if evP.Type != evdev.EV_KEY {
		// fmt.Printf(" skipping %s\n", evP.String())
		err = ew.WriteOne(evP)
		if err != nil {
			return err
		}
		return nil
	}
	switch evP.Value {
	case UP:
		err = state.HandleUpChar(*evP)
	case DOWN:
		err = state.HandleDownChar(*evP)
	case REPEAT:
		fmt.Printf(" skipping (repeat): %s\n", evP.String())
		return nil
	default:
		return fmt.Errorf("Received %d. Expected UP or DOWN", evP.Value)
	}
	if err != nil {
		return err
	}
	return nil
}

func NewState(maxLength int, ew EventWriter, allCombos []*Combo) *State {
	s := State{
		outDev:             ew,
		allCombos:          allCombos,
		minOverlapDuration: 80 * time.Millisecond,
	}
	s.buf = make([]Event, 0, maxLength)
	s.fakeActiverTimerNextTime = maxTime
	return &s
}

// Maximum possible time.Time value
var maxTime = time.Unix(1<<63-62135596801, 999999999)

type State struct {
	buf                      []Event
	allCombos                []*Combo
	downKeysWritten          []*Combo
	swallowKeys              []KeyCode
	minOverlapDuration       time.Duration
	outDev                   EventWriter
	activeTimer              <-chan time.Time // fires N milliseconds after the last key-down-event.
	fakeActiveTimer          bool             // In tests the activeTimer will be faked by reading the time of the next event.
	fakeActiverTimerNextTime time.Time        // The next the fakeActiveTimer event will be fired.
}

func (state *State) Eval(time syscall.Timeval, reason string) error {
	fmt.Printf("Eval [%s] %s\n", reason, state.String())
	if len(state.buf) == 2 &&
		state.buf[0].Code == state.buf[1].Code &&
		state.buf[0].Value == DOWN && state.buf[1].Value == UP {
		newSwallowKeys := make([]KeyCode, 0, len(state.swallowKeys))
		doSwallow := false
		for _, key := range state.swallowKeys {
			if state.buf[0].Code == key && state.buf[1].Code == key {
				doSwallow = true
				continue
			}
			newSwallowKeys = append(newSwallowKeys, key)
		}
		if doSwallow {
			state.swallowKeys = newSwallowKeys
			fmt.Printf("  SwallowKeys: %s\n", state.String())
			state.buf = nil
			return nil
		}
		state.FlushBuffer("Eval>up-down-of-singlechar")
		return nil
	}
	combos := state.allCombos
	codes := make([]evalResult, 0, len(combos))
	for _, combo := range combos {
		code, msg, err := state.EvalCombo(combo, time)
		if err != nil {
			return fmt.Errorf("failed to eval combo: %w", err)
		}
		fmt.Printf("  EvalCombo %s %s: %s\n", combo.String(), code, msg)
		codes = append(codes, code)
	}
	// Handle WriteUpKeys first
	found := false
	for i, code := range codes {
		if code != WriteUpKeys {
			continue
		}
		found = true
		combo := combos[i]
		err := state.WriteComboDownKeysNew(combo)
		if err != nil {
			return fmt.Errorf("failed to write combo (down): %w", err)
		}
		err = state.WriteComboUpKeysNew(combo)
		if err != nil {
			return fmt.Errorf("failed to write combo (up): %w", err)
		}
	}
	if found {
		return nil
	}

	// Handle AllDownKeysSeen
	for i := range codes {
		if codes[i] != AllDownKeysSeen {
			continue
		}
		combo := combos[i]
		found = true
		if slices.Contains(state.downKeysWritten, combo) {
			continue
		}
		err := state.WriteComboDownKeysNew(combo)
		if err != nil {
			return fmt.Errorf("failed to write combo (AllDownKeysSeen): %w", err)
		}
		state.downKeysWritten = append(state.downKeysWritten, combo)
	}
	if found {
		return nil
	}

	// Handle ComboNotFinishedCode
	for _, code := range codes {
		if code != ComboNotFinished {
			continue
		}
		found = true
	}
	if found {
		return nil
	}

	// no match. Flush buffer.
	return state.FlushBuffer("Eval>No-match")
}

type evalResult string

var (
	NoMatch                          evalResult = "NoMatch"
	Error                            evalResult = "Error"
	ComboNotFinished                 evalResult = "ComboNotFinished"
	WriteUpKeys                      evalResult = "WriteUpKeys"
	AllDownKeysSeen                  evalResult = "AllDownKeysSeen" // do not write the out-keys yet. It could be an unintended overlap.
	AllDownKeysSeenAndAlreadyWritten evalResult = "AllDownKeysSeenAndAlreadyWritten"
)

func (state *State) EvalCombo(combo *Combo, currTime syscall.Timeval) (evalResult, string, error) {
	// check if all down-keys are seen, and in the same order.
	seenDown := make([]KeyCode, 0, len(combo.Keys))
	seenUp := make([]KeyCode, 0, len(combo.Keys))
	var lastDownEvent *evdev.InputEvent
	var firstUpEvent *evdev.InputEvent
	var unknownKey *KeyCode
	for _, ev := range state.buf {
		if !slices.Contains(combo.Keys, ev.Code) {
			unknownKey = &ev.Code
			break
		}
		switch ev.Value {
		case DOWN:
			lastDownEvent = &ev
			seenDown = append(seenDown, ev.Code)
		case UP:
			if firstUpEvent == nil {
				firstUpEvent = &ev
			}
			seenUp = append(seenUp, ev.Code)
		default:
			return Error, "", fmt.Errorf("unexpected value %d", ev.Value)
		}
	}
	if len(seenDown) == 0 {
		return NoMatch, "No down-keys seen", nil
	}
	if unknownKey != nil {
		return NoMatch, "Unknown key in buffer: " + keyToString(*unknownKey), nil
	}

	for i, key := range combo.Keys {
		if i >= len(seenDown) {
			// Not all down-keys are seen.
			return ComboNotFinished,
				fmt.Sprintf("seenDown %s", SliceOfKeysToString(seenDown)),
				nil
		}
		if seenDown[i] != key {
			// Order is wrong. For example "J F" instead of "F J".
			return NoMatch, fmt.Sprintf("Order is wrong %s", SliceOfKeysToString(seenDown)), nil
		}
	}

	// All down-keys are seen.

	if firstUpEvent != nil &&
		syscallTimevalToTime(lastDownEvent.Time).Before(syscallTimevalToTime(firstUpEvent.Time)) &&
		lastDownEvent.Code != firstUpEvent.Code {

		overlapDuration := timeSub(*&lastDownEvent.Time, firstUpEvent.Time)
		if overlapDuration < 40*time.Millisecond {
			return NoMatch, fmt.Sprintf("Overlap too short %s", overlapDuration), nil
		}
	}
	age := timeSub(lastDownEvent.Time, currTime)
	minAge := 140 * time.Millisecond
	if age < minAge {
		// All in-down-keys are seen. But wait some milliseconds before writing the out-down-keys.
		return ComboNotFinished,
			fmt.Sprintf("All down seen, but too young (lastDown..currTime minAge %s): %s", minAge.String(), age.String()), nil
	}
	if len(seenUp) > 0 {
		return WriteUpKeys, fmt.Sprintf("WriteUpKeys. Finished %s", SliceOfKeysToString(seenUp)), nil
	}
	if slices.Contains(state.downKeysWritten, combo) {
		return AllDownKeysSeenAndAlreadyWritten, "", nil
	}
	return AllDownKeysSeen, "All down seen. Write the out-down-keys", nil
}

// AfterTimer gets called N milliseconds after the last key-down-event.
// If all down-keys got pressed (and held down, no up-keys were seen yet),
// then we need to fire the down events after some time.
func (state *State) AfterTimer() error {
	panic("AfterTimer not during testsssssssssssss")
	timeval := syscall.Timeval{}
	syscall.Gettimeofday(&timeval)
	return state.Eval(timeval, "timer")
}

func (state *State) fakeAfterTimerFunc(time time.Time) error {
	return state.Eval(timeToSyscallTimeval(time), "timer")
}

func (state *State) WriteComboDownKeysNew(combo *Combo) error {
	if slices.Contains(state.downKeysWritten, combo) {
		// Down-Keys have already been written.
		// Don't write them again.
		return nil
	}
	// We don't remove events from the buffer. This gets done,
	// when the corresponding up-keys are seen.
	state.WriteCombo(combo, state.buf[0].Time, DOWN)
	return nil
}

func (state *State) WriteComboUpKeysNew(combo *Combo) error {
	if slices.Contains(state.downKeysWritten, combo) {
		state.downKeysWritten = removeFromSlice(state.downKeysWritten, combo)
	}
	seenUp := make([]KeyCode, 0, len(combo.Keys))
	for _, ev := range state.buf {
		if slices.Contains(combo.Keys, ev.Code) && ev.Value == UP {
			seenUp = append(seenUp, ev.Code)
		}
	}
	missingUp := make([]KeyCode, 0, len(combo.Keys))
	for _, key := range combo.Keys {
		if !slices.Contains(seenUp, key) {
			missingUp = append(missingUp, key)
		}
	}
	state.swallowKeys = append(state.swallowKeys, missingUp...)
	// This is the final action for this combo.
	// Now the corresponding keys in the buffer get removed.
	newBuf := make([]Event, 0, len(state.buf))
	for _, ev := range state.buf {
		if slices.Contains(seenUp, ev.Code) {
			continue
		}
		newBuf = append(newBuf, ev)
	}
	state.WriteCombo(combo, state.buf[0].Time, UP)
	state.buf = newBuf
	return nil
}

type upDownValue = int32

func (state *State) WriteCombo(combo *Combo, time syscall.Timeval, value upDownValue) error {
	// first match. Use that timestamp to write out the combo.
	for _, outKey := range combo.OutKeys {
		err := state.WriteEvent(evdev.InputEvent{
			Time:  time,
			Type:  evdev.EV_KEY,
			Code:  outKey,
			Value: value,
		}, fmt.Sprintf("WriteCombo %s %s", eventValueToString[value], combo.String()))
		if err != nil {
			return fmt.Errorf("failed to write DOWN event: %w", err)
		}
	}
	return nil
}

func (state *State) WriteEvent(ev Event, reason string) error {
	fmt.Printf("  write %s %s\n", eventToString(&ev), reason)
	err := state.outDev.WriteOne(&ev)
	return errors.Join(err, state.outDev.WriteOne(&Event{
		Time: ev.Time,
		Type: evdev.EV_SYN,
		Code: evdev.SYN_REPORT,
	}))
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
			ret = append(ret, fmt.Sprintf("(%s)", timeSub(prev.Time, ev.Time).String()))
		}
		ret = append(ret, eventToString(&ev))
		prev = &ev
	}
	if len(ret) == 0 {
		ret = append(ret, "buf is empty")
	}
	ret = []string{strings.Join(ret, " ")}
	downKeys := make([]string, 0, len(state.downKeysWritten))
	for _, combo := range state.downKeysWritten {
		downKeys = append(downKeys, combo.String())
	}
	ret = append(ret, fmt.Sprintf("downKeysWritten: %v", downKeys))
	ret = append(ret, fmt.Sprintf("swallowKeys: %v", SliceOfKeysToString(state.swallowKeys)))
	return strings.Join(ret, " ")
}

// The buffered events don't match a combo.
// Write out the buffered events.
func (state *State) FlushBuffer(reason string) error {
	for _, bufEvent := range state.buf {
		if err := state.WriteEvent(bufEvent, reason+">FlushBuffer"); err != nil {
			return err
		}
	}
	state.buf = nil
	state.activeTimer = nil
	state.fakeActiverTimerNextTime = maxTime
	return nil
}

func (state *State) FlushBufferAndWriteEvent(ev Event, reason string) error {
	err := state.FlushBuffer(reason + ">FlushBufferAndWriteEvent-flushBuff")
	if err != nil {
		return err
	}
	return state.WriteEvent(ev, reason+">FlushBufferAndWriteEvent-writeE")
}

func (state *State) HandleUpChar(
	ev Event,
) error {
	state.buf = append(state.buf, ev)
	return state.Eval(ev.Time, "up")
}

func (state *State) HandleDownChar(
	ev Event,
) error {
	timeoutAfterDownDuration := 150 * time.Millisecond
	if state.fakeActiveTimer {
		// For testing.
		state.fakeActiverTimerNextTime = syscallTimevalToTime(ev.Time).Add(timeoutAfterDownDuration)
	} else {
		state.activeTimer = time.After(timeoutAfterDownDuration)
	}

	state.buf = append(state.buf, ev)
	return state.Eval(ev.Time, "down")
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
		if eventToSkip(ev) {
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
	timeoutSeconds := 5 * time.Second
	fmt.Printf("Grabbing %s\n", targetName)
	fmt.Printf("Do not type for %s to terminate.\n", timeoutSeconds)
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
			if duration > timeoutSeconds {
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
		if eventToSkip(ev) {
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
			s = eventToString(ev)
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

func eventToString(ev *Event) string {
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

	name = name + eventValueToShortString[ev.Value]
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
			return nil, fmt.Errorf("csv to slice failed: %w", err)
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
		if eventToSkip(&ev) {
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
		if err != nil {
			return nil, fmt.Errorf("csvlineToEvent failed: %w", err)
		}
		// fmt.Printf("<<| %s\n", eventToString(&ev))
		return &ev, nil
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

func removeFromSlice[T comparable](s []T, elem T) []T {
	newSlice := make([]T, 0, len(s))
	for i := range s {
		if s[i] == elem {
			continue
		}
		newSlice = append(newSlice, s[i])
	}
	return newSlice
}

func eventToSkip(ev *Event) bool {
	if ev.Type == evdev.EV_SYN {
		return true
	}
	if ev.Type == evdev.EV_MSC && ev.Code == evdev.MSC_SCAN {
		return true
	}
	return false
}
