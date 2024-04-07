package main

import (
	"fmt"
	"io"
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
	fmt.Printf("ReadOne %s len: %d\n", eventToCsvLine(ev), len(rfs.s))
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

var asCombo = `1711354959;655837;EV_KEY;KEY_A;down
1711354959;815829;EV_KEY;KEY_A;up
1711354959;999756;EV_KEY;KEY_S;down
1711354960;127830;EV_KEY;KEY_S;up
1711354960;591917;EV_KEY;KEY_S;down
1711354960;711880;EV_KEY;KEY_S;up
1711354960;887889;EV_KEY;KEY_A;down
1711354961;39896;EV_KEY;KEY_A;up
1711354962;536044;EV_KEY;KEY_A;down
1711354962;591749;EV_KEY;KEY_S;down
1711354962;647719;EV_KEY;KEY_A;up
1711354962;695617;EV_KEY;KEY_S;up
1711354963;928070;EV_KEY;KEY_A;down
1711354963;935975;EV_KEY;KEY_S;down
1711354964;327860;EV_KEY;KEY_S;up
1711354964;359906;EV_KEY;KEY_A;up
`

var asdfTestEvents = `1712518531;862966;EV_KEY;KEY_A;down
1712518532;22233;EV_KEY;KEY_A;up
1712518532;478346;EV_KEY;KEY_S;down
1712518532;637660;EV_KEY;KEY_S;up
1712518533;35798;EV_KEY;KEY_D;down
1712518533;132219;EV_KEY;KEY_D;up
1712518533;948232;EV_KEY;KEY_F;down
1712518534;116984;EV_KEY;KEY_F;up
`

func Test_manInTheMiddle_asdf_noMatch(t *testing.T) {
	ew := writeToSlice{}
	er, err := NewReadFromSlice(asdfTestEvents)
	require.Nil(t, err)
	allCombos := []Combo{
		{
			Keys:    []KeyCode{evdev.KEY_G, evdev.KEY_H},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	}
	err = manInTheMiddle(er, &ew, allCombos)
	require.ErrorIs(t, io.EOF, err)
	csv := eventsToCsv(ew.s)
	require.Equal(t, asdfTestEvents, csv)
}

func Test_manInTheMiddle_asdf_ComboButNoMatch(t *testing.T) {
	ew := writeToSlice{}
	er, err := NewReadFromSlice(asdfTestEvents)
	require.Nil(t, err)
	allCombos := []Combo{
		{
			Keys:    []KeyCode{evdev.KEY_A, evdev.KEY_F},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	}
	err = manInTheMiddle(er, &ew, allCombos)
	require.ErrorIs(t, io.EOF, err)
	csv := eventsToCsv(ew.s)
	require.Equal(t, asdfTestEvents, csv)
}
