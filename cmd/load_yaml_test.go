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
