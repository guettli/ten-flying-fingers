package tff

import (
	"fmt"
	"os"
)

func PrintMain(path string) error {
	sourceDev, err := GetDeviceFromPath(path)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	err = printEvents(sourceDev)
	if err != nil {
		fmt.Println(err.Error())
	}
	return nil
}
