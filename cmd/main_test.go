package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/holoplot/go-evdev"
	"github.com/stretchr/testify/require"
)

type writeToSlice struct {
	s []Event
}

func (wts *writeToSlice) WriteOne(ev *evdev.InputEvent) error {
	wts.s = append(wts.s, *ev)
	return nil
}

func (wts *writeToSlice) requireEqual(t *testing.T, expectedShort string) {
	t.Helper()
	actualShort, err := csvToShortCsv(eventsToCsv(wts.s))
	if err != nil {
		t.Fatal(err.Error())
	}
	var e []string
	for _, line := range strings.Split(expectedShort, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		e = append(e, line)

	}
	expectedShort = strings.Join(e, "\n")
	require.Equal(t, expectedShort, actualShort)
}

func csvToShortCsv(csv string) (string, error) {
	var e []string
	for _, line := range strings.Split(csv, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line, err := csvLineToShortLine(line)
		if err != nil {
			return "", err
		}
		e = append(e, line)
	}
	return strings.Join(e, "\n"), nil
}

var csvLineToShortLineRegex = regexp.MustCompile(`^\d+;\d+;EV_KEY;KEY_(\w+);(\w+)$`)

func csvLineToShortLine(csvLine string) (string, error) {
	matches := csvLineToShortLineRegex.FindStringSubmatch(csvLine)
	if matches == nil || len(matches) != 3 {
		return "", fmt.Errorf("failed to parse csvLine %s %+v", csvLine, matches)
	}
	return fmt.Sprintf("%s-%s", matches[1], matches[2]), nil
}

var _ = EventWriter(&writeToSlice{})

type readFromSlice struct {
	s []evdev.InputEvent
}

func (rfs *readFromSlice) ReadOne() (*Event, error) {
	if len(rfs.s) == 0 {
		return nil, io.EOF
	}
	ev := rfs.s[0]
	rfs.s = rfs.s[1:]
	return &ev, nil
}

func (rfs *readFromSlice) loadCSV(csvString string) error {
	s, err := csvToSlice(csvString)
	rfs.s = s
	return err
}

func NewReadFromSlice(csvString string) (*readFromSlice, error) {
	rfs := readFromSlice{}
	err := rfs.loadCSV(csvString)
	return &rfs, err
}

var _ = EventReader(&readFromSlice{})

var asdfTestEvents = `1712518531;862966;EV_KEY;KEY_A;down
1712518532;22233;EV_KEY;KEY_A;up
1712518532;478346;EV_KEY;KEY_S;down
1712518532;637660;EV_KEY;KEY_S;up
1712518533;35798;EV_KEY;KEY_D;down
1712518533;132219;EV_KEY;KEY_D;up
1712518533;948232;EV_KEY;KEY_F;down
1712518534;116984;EV_KEY;KEY_F;up
`

func Test_manInTheMiddle_noMatch(t *testing.T) {
	f := func(allCombos []Combo) {
		ew := writeToSlice{}
		er, err := NewReadFromSlice(asdfTestEvents)
		require.Nil(t, err)
		err = manInTheMiddle(er, &ew, allCombos, false)
		require.ErrorIs(t, err, io.EOF)
		csv := eventsToCsv(ew.s)
		require.Equal(t, asdfTestEvents, csv)
	}
	f([]Combo{
		{
			Keys:    []KeyCode{evdev.KEY_A, evdev.KEY_F},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	})

	f([]Combo{
		{
			Keys:    []KeyCode{evdev.KEY_G, evdev.KEY_H},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	})

	f([]Combo{
		{
			Keys:    []KeyCode{evdev.KEY_G, evdev.KEY_H},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
		{
			Keys:    []KeyCode{evdev.KEY_A, evdev.KEY_F},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	})
}

func Test_manInTheMiddle_asdf_ComboWithMatch(t *testing.T) {
	allCombos := []Combo{
		{
			Keys:    []KeyCode{evdev.KEY_A, evdev.KEY_F},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
		{
			Keys:    []KeyCode{evdev.KEY_A, evdev.KEY_J},
			OutKeys: []KeyCode{evdev.KEY_Y},
		},
	}
	f := func(input string, expectedOutput string) {
		ew := writeToSlice{}
		er, err := NewReadFromSlice(input)
		require.Nil(t, err)
		err = manInTheMiddle(er, &ew, allCombos, false)
		require.ErrorIs(t, err, io.EOF)
		ew.requireEqual(t, expectedOutput)
	}
	f(`
			1712519053;127714;EV_KEY;KEY_B;down
			1712519053;149844;EV_KEY;KEY_B;up
			1712519053;827714;EV_KEY;KEY_F;down
			1712519053;849844;EV_KEY;KEY_A;down
			1712519054;320867;EV_KEY;KEY_A;up
			1712519054;321153;EV_KEY;KEY_F;up
			1712519055;127714;EV_KEY;KEY_C;down
			1712519055;149844;EV_KEY;KEY_C;up
			`,
		`
			B-down
			B-up
			X-down
			X-up
			C-down
			C-up
			`)
	f(`
			1716752333;203961;EV_KEY;KEY_A;down
			1716752333;327486;EV_KEY;KEY_A;up
			`,
		`
			A-down
			A-up
			`,
	)
	f(`
			1712519053;827714;EV_KEY;KEY_F;down
			1712519053;849844;EV_KEY;KEY_A;down
			1712519054;320867;EV_KEY;KEY_A;up
			1712519054;321153;EV_KEY;KEY_F;up
			`,
		`
			X-down
			X-up
			`,
	)

	// short overlap between F-down and A-up.
	// This is A followed by F, not a combo.
	f(`
			1712519053;827714;EV_KEY;KEY_A;down
			1712519054;320840;EV_KEY;KEY_F;down
			1712519054;320860;EV_KEY;KEY_A;up
			1712519054;321153;EV_KEY;KEY_F;up
			`,
		`
			A-down
			F-down
			A-up
			F-up
			`)
}

func Test_manInTheMiddle_asdf_TwoJoinedCombos(t *testing.T) {
	// A-down, F-down, F-up (emit x), J-down, F-up (emit y)
	allCombos := []Combo{
		{
			Keys:    []KeyCode{evdev.KEY_A, evdev.KEY_F},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
		{
			Keys:    []KeyCode{evdev.KEY_A, evdev.KEY_J},
			OutKeys: []KeyCode{evdev.KEY_Y},
		},
	}
	f := func(input string, expectedOutput string) {
		ew := writeToSlice{}
		er, err := NewReadFromSlice(input)
		require.Nil(t, err)
		err = manInTheMiddle(er, &ew, allCombos, false)
		require.ErrorIs(t, err, io.EOF)
		ew.requireEqual(t, expectedOutput)
	}
	f(`
			1716752333;000000;EV_KEY;KEY_A;down
			1716752333;100000;EV_KEY;KEY_F;down
			1716752333;200000;EV_KEY;KEY_F;up
			1716752333;300000;EV_KEY;KEY_J;down
			1716752333;400000;EV_KEY;KEY_J;up
			1716752333;500000;EV_KEY;KEY_A;up
			`,
		`
			X-down
			X-up
			`,
	)
}
