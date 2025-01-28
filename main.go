package main

import (
	"fmt"
	"os"

	"github.com/guettli/tff/pkg/tff"
)

func main() {
	err := tff.MyMain()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
