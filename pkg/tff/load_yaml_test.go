package tff

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
		inString        string
		expectedKeyCode KeyCode
		expectedError   error
	}{
		{"x", evdev.KEY_X, nil},
		{"1", evdev.KEY_1, nil},
		{"capslock", evdev.KEY_CAPSLOCK, nil},
		{"X", 0, OnlyLowerCaseAllowedErr},
		{"ü", 0, UnknownKeyErr},
	}
	for _, tt := range tests {
		got, err := wordToKeyCode(tt.inString)
		if tt.expectedError != nil {
			require.ErrorIs(t, err, tt.expectedError)
		} else {
			require.Nil(t, err)
		}
		require.Equal(t, tt.expectedKeyCode, got)
	}
}

func TestLoadYamlFromBytes_ok(t *testing.T) {
	yamlString := `combos:
  - keys: f j
    outKeys: a b c
`
	expected := []*Combo{
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
	}
	actual, err := LoadYamlFromBytes([]byte(yamlString))
	require.Nil(t, err)
	require.Equal(t, expected, actual)
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
    outKeys: a b key_not_existing
`,
			`failed to get key "key_not_existing"`,
		},
	}
	for _, tt := range tests {
		_, err := LoadYamlFromBytes([]byte(tt.yamlString))
		require.ErrorContains(t, err, tt.expected)
	}
}
