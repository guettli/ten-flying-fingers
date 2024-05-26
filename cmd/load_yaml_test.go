package main

import (
	"testing"

	"github.com/holoplot/go-evdev"
	"github.com/stretchr/testify/require"
)

func Test_runeToKeyCode(t *testing.T) {
	type args struct {
		r rune
	}
	tests := []struct {
		inRune          rune
		expectedKeyCode KeyCode
		expectedError   error
	}{
		{'x', evdev.KEY_X, nil},
		{'1', evdev.KEY_1, nil},
		{'X', 0, OnlyLowerCaseAllowedErr},
		{'ü', 0, UnknownKeyErr},
	}
	for _, tt := range tests {
		got, err := runeToKeyCode(tt.inRune)
		if tt.expectedError != nil {
			require.ErrorIs(t, err, tt.expectedError)
		} else {
			require.Nil(t, err)
		}
		require.Equal(t, tt.expectedKeyCode, got)
	}
}

func TestLoadYamlFromBytes_ok(t *testing.T) {
	tests := []struct {
		yamlString string
		expected   []Combo
	}{
		{
			`combos:
  - keys: f  KEY_J
    outKeys: a b  KEY_C
`,
			[]Combo{
				{
					Keys: []evdev.EvCode{
						evdev.KEY_F,
						evdev.KEY_J,
					},
					OutKeys: []evdev.EvCode{
						evdev.KEY_A,
						evdev.KEY_B,
						evdev.KEY_C,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		actual, err := LoadYamlFromBytes([]byte(tt.yamlString))
		require.Nil(t, err)
		require.Equal(t, tt.expected, actual)
	}
}

func TestLoadYamlFromBytes_fail(t *testing.T) {
	tests := []struct {
		yamlString string
		expected   string
	}{
		{
			`combos:
  - keys: f j
  - outKeys: a b c
`,
			`empty list in 'outKeys' is not allowed.`,
		},
		{
			`combos:
  - outKeys: a b c
`,
			`empty list in 'keys' is not allowed.`,
		},
		{
			`combos
  - keys: f j
  - outKeys: a b c
`,
			"mapping values are not allowed in this context",
		},
		{
			`combos:
  - keys: f j
    outKeys: a b KEY_not_existing
`,
			`failed to get key "KEY_not_existing"`,
		},
	}
	for _, tt := range tests {
		_, err := LoadYamlFromBytes([]byte(tt.yamlString))
		require.ErrorContains(t, err, tt.expected)
	}
}
