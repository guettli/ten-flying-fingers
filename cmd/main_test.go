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

func AssertComboInputOutput(t *testing.T, input string, expectedOutput string, allCombos []*Combo) {
	t.Helper()
	ew := writeToSlice{}
	er, err := NewReadFromSlice(input)
	require.Nil(t, err)
	err = manInTheMiddle(er, &ew, allCombos, true, true)
	require.ErrorIs(t, err, io.EOF)
	ew.requireEqual(t, expectedOutput)
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

var asdfTestEvents = `1712500001;862966;EV_KEY;KEY_A;down
1712500002;22233;EV_KEY;KEY_A;up
1712500002;478346;EV_KEY;KEY_S;down
1712500002;637660;EV_KEY;KEY_S;up
1712500003;35798;EV_KEY;KEY_D;down
1712500003;132219;EV_KEY;KEY_D;up
1712500003;948232;EV_KEY;KEY_F;down
1712500004;116984;EV_KEY;KEY_F;up
`

var fjkCombos = []*Combo{
	{
		Keys:    []KeyCode{evdev.KEY_F, evdev.KEY_J},
		OutKeys: []KeyCode{evdev.KEY_X},
	},
	{
		Keys:    []KeyCode{evdev.KEY_F, evdev.KEY_K},
		OutKeys: []KeyCode{evdev.KEY_Y},
	},
}

func Test_manInTheMiddle_noMatch(t *testing.T) {
	f := func(allCombos []*Combo) {
		ew := writeToSlice{}
		er, err := NewReadFromSlice(asdfTestEvents)
		require.Nil(t, err)
		err = manInTheMiddle(er, &ew, allCombos, false, true)
		require.ErrorIs(t, err, io.EOF)
		csv := eventsToCsv(ew.s)
		require.Equal(t, asdfTestEvents, csv)
	}
	f([]*Combo{
		{
			Keys:    []KeyCode{evdev.KEY_A, evdev.KEY_F},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	})

	f([]*Combo{
		{
			Keys:    []KeyCode{evdev.KEY_G, evdev.KEY_H},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	})

	f([]*Combo{
		{
			Keys:    []KeyCode{evdev.KEY_G, evdev.KEY_H},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
		{
			Keys:    []KeyCode{evdev.KEY_A, evdev.KEY_K},
			OutKeys: []KeyCode{evdev.KEY_X},
		},
	})
}

// //////////////////////////////////////////////
func Test_manInTheMiddle_NoMatch_JustKeys(t *testing.T) {
	AssertComboInputOutput(t, `
	1712500000;000000;EV_KEY;KEY_B;down
	1712500000;020000;EV_KEY;KEY_B;up
	1712500000;700000;EV_KEY;KEY_F;down
	1712500000;720000;EV_KEY;KEY_F;up
	1712500001;100000;EV_KEY;KEY_J;down
	1712500001;110000;EV_KEY;KEY_J;up
	1712500001;800000;EV_KEY;KEY_C;down
	1712500001;900000;EV_KEY;KEY_C;up
	`,
		`
	B-down
	B-up
	F-down
	F-up
	J-down
	J-up
	C-down
	C-up
	`, fjkCombos)
}

func Test_manInTheMiddle_TwoCombos_WithOneEmbrachingMatch(t *testing.T) {
	AssertComboInputOutput(t, `
	1712500000;000000;EV_KEY;KEY_B;down
	1712500000;020000;EV_KEY;KEY_B;up
	1712500000;700000;EV_KEY;KEY_F;down
	1712500000;720000;EV_KEY;KEY_J;down
	1712500001;100000;EV_KEY;KEY_J;up
	1712500001;110000;EV_KEY;KEY_F;up
	1712500001;800000;EV_KEY;KEY_C;down
	1712500001;900000;EV_KEY;KEY_C;up
	`,
		`
	B-down
	B-up
	X-down
	X-up
	C-down
	C-up
	`, fjkCombos)
}

func Test_manInTheMiddle_SingleCombo_OneEmbrachingMatch(t *testing.T) {
	AssertComboInputOutput(t, `
	1712500003;827714;EV_KEY;KEY_F;down
	1712500003;849844;EV_KEY;KEY_J;down
	1712500004;320867;EV_KEY;KEY_J;up
	1712500004;321153;EV_KEY;KEY_F;up
	`,
		`
	X-down
	X-up
	`,
		fjkCombos)
}

func Test_manInTheMiddle_ComboWithMatch_CrossRhyme(t *testing.T) {
	AssertComboInputOutput(t, `
	1712500000;700000;EV_KEY;KEY_F;down
	1712500000;720000;EV_KEY;KEY_J;down
	1712500001;100000;EV_KEY;KEY_F;up
	1712500001;110000;EV_KEY;KEY_J;up
	1712500001;800000;EV_KEY;KEY_C;down
	1712500001;900000;EV_KEY;KEY_C;up
	`,
		`
	X-down
	X-up
	C-down
	C-up
	`, fjkCombos)
}

func Test_manInTheMiddle_ComboWithMatch_SingleUpDown(t *testing.T) {
	AssertComboInputOutput(t, `
	1716752333;203961;EV_KEY;KEY_F;down
	1716752333;327486;EV_KEY;KEY_F;up
	`,
		`
	F-down
	F-up
	`,
		fjkCombos)
}

func Test_manInTheMiddle_ComboWithMatch_OverlapNoCombo(t *testing.T) {
	// short overlap between K-down and F-up.
	// This is F followed by K, not a combo.
	AssertComboInputOutput(t, `
	1712500003;827714;EV_KEY;KEY_F;down
	1712500004;320840;EV_KEY;KEY_J;down
	1712500004;320860;EV_KEY;KEY_F;up
	1712500004;321153;EV_KEY;KEY_J;up
	`,
		`
	F-down
	J-down
	F-up
	J-up
	`, fjkCombos)
}

func Test_manInTheMiddle_WithoutMatch(t *testing.T) {
	AssertComboInputOutput(t, `
	1712500000;700000;EV_KEY;KEY_K;down
	1712500000;820000;EV_KEY;KEY_K;up
	1712500000;830000;EV_KEY;KEY_F;down
	1712500000;840000;EV_KEY;KEY_F;up
	`,
		`
	K-down
	K-up
	F-down
	F-up
	`, fjkCombos)
}

func Test_manInTheMiddle_TwoComboWithSingleMatch(t *testing.T) {
	AssertComboInputOutput(t, `
	1712500000;000000;EV_KEY;KEY_B;down
	1712500000;020000;EV_KEY;KEY_B;up
	1712500000;700000;EV_KEY;KEY_F;down
	1712500000;720000;EV_KEY;KEY_J;down
	1712500001;100000;EV_KEY;KEY_J;up
	1712500001;110000;EV_KEY;KEY_F;up
	1712500001;800000;EV_KEY;KEY_C;down
	1712500001;900000;EV_KEY;KEY_C;up
	`,
		`
	B-down
	B-up
	X-down
	X-up
	C-down
	C-up
	`, fjkCombos)
}

func Test_manInTheMiddle_TwoEmbrachingCombosWithMatch(t *testing.T) {
	AssertComboInputOutput(t, `
	1716752333;000000;EV_KEY;KEY_F;down
	1716752333;100000;EV_KEY;KEY_J;down
	1716752333;400000;EV_KEY;KEY_J;up
	1716752333;600000;EV_KEY;KEY_K;down
	1716752333;800000;EV_KEY;KEY_K;up
	1716752334;000000;EV_KEY;KEY_F;up
	`,
		`
	X-down
	X-up
	Y-down
	Y-up
	`,
		fjkCombos)
}

func Test_manInTheMiddle_TwoJoinedCombos_FirstKeyDownUntilEnd(t *testing.T) {
	AssertComboInputOutput(t, `
	1716752333;000000;EV_KEY;KEY_F;down
	1716752333;100000;EV_KEY;KEY_J;down
	1716752333;400000;EV_KEY;KEY_J;up
	1716752333;600000;EV_KEY;KEY_K;down
	1716752333;800000;EV_KEY;KEY_K;up
	1716752334;000000;EV_KEY;KEY_F;up
	`,
		`
	X-down
	X-up
	Y-down
	Y-up
	`,
		fjkCombos)
}

func Test_manInTheMiddle_ComboWithMatch_NoPanic(t *testing.T) {
	// This test is to ensure that no panic happens.
	// Output could be different.
	AssertComboInputOutput(t, `
	1712500000;000000;EV_KEY;KEY_F;down
	1712500000;064000;EV_KEY;KEY_K;down
	1712500000;128000;EV_KEY;KEY_F;up
	1712500000;144000;EV_KEY;KEY_J;down
	1712500000;208000;EV_KEY;KEY_K;up
	1712500000;224000;EV_KEY;KEY_F;down
`,
		// The input is quite crazy. This tests ensures that no panic happens.
		// Changes are allowed to alter the output.
		`
	Y-down
	Y-up
	J-down
	F-down
	`, fjkCombos)
}

//////////////////////////////

var orderedCombos = []*Combo{
	{
		Keys:    []KeyCode{evdev.KEY_F, evdev.KEY_J},
		OutKeys: []KeyCode{evdev.KEY_X},
	},
	{
		Keys:    []KeyCode{evdev.KEY_J, evdev.KEY_F},
		OutKeys: []KeyCode{evdev.KEY_A},
	},
	{
		Keys:    []KeyCode{evdev.KEY_F, evdev.KEY_K},
		OutKeys: []KeyCode{evdev.KEY_Y},
	},
	{
		Keys:    []KeyCode{evdev.KEY_J, evdev.KEY_K},
		OutKeys: []KeyCode{evdev.KEY_B},
	},
}

func Test_orderedCombos(t *testing.T) {
	AssertComboInputOutput(t,
		`
	1712500000;000000;EV_KEY;KEY_F;down
	1712500000;060000;EV_KEY;KEY_J;down
	1712500000;120000;EV_KEY;KEY_F;up
	1712500000;200000;EV_KEY;KEY_J;up

	1712500001;000000;EV_KEY;KEY_J;down
	1712500001;060000;EV_KEY;KEY_F;down
	1712500001;120000;EV_KEY;KEY_J;up
	1712500001;200000;EV_KEY;KEY_F;up
	`,
		`
		X-down
		X-up
		A-down
		A-up
	`,
		orderedCombos)
}
