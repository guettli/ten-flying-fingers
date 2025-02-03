package tff

import (
	"fmt"
	"time"

	"github.com/holoplot/go-evdev"
)

func Csv(sourceDev *evdev.InputDevice) error {
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
