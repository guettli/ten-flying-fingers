package main

import (
	"fmt"
	"io"
	"testing"
	"time"

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

var asdfNoCombo = `1712516686;34146;EV_KEY;KEY_S;down
1712516686;166940;EV_KEY;KEY_S;up
1712516686;747419;EV_KEY;KEY_D;down
1712516686;879558;EV_KEY;KEY_D;up
1712516687;527079;EV_KEY;KEY_F;down
1712516687;686178;EV_KEY;KEY_F;up
`

func Test_manInTheMiddle(t *testing.T) {
	ew := writeToSlice{}
	er, err := NewReadFromSlice(asdfNoCombo)
	require.Nil(t, err)
	timeout := time.After(1 * time.Second)
	done := make(chan error)
	allCombos := []Combo{
		{
			Keys:    []KeyCode{evdev.KEY_G, evdev.KEY_H},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	}
	go func() {
		done <- manInTheMiddle(er, &ew, allCombos)
	}()

	select {
	case <-timeout:
		t.Fatal("Test didn't finish in time")
	case err = <-done:
	}
	require.ErrorIs(t, io.EOF, err)
	csv := eventsToCsv(ew.s)
	require.Equal(t, asdfNoCombo, csv)
}
