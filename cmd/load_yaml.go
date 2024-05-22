package main

import (
	"fmt"
	"os"
	"unicode"

	"github.com/holoplot/go-evdev"
	"gopkg.in/yaml.v3"
)

type Yaml struct {
	Combos []struct {
		Keys    string `yaml:"keys"`
		OutKeys string `yaml:"outKeys"`
	} `yaml:"combos"`
}

func LoadYamlFile(yamlFile string) ([]Combo, error) {
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml config from %q: %w", yamlFile, err)
	}
	y := Yaml{}
	err = yaml.Unmarshal(data, &y)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", yamlFile, err)
	}
	//_ := make([]Combo, 0, len(y.Combos))
	return nil, nil
}

func wordToKeyCode(word string) (KeyCode, error) {
	runes := []rune(word)
	if len(runes) == 1 {
		return runeToKeyCode(runes[0])
	}
	return 0, nil
}

var (
	OnlyLowerCaseAllowedErr = fmt.Errorf("only lower case characters are allowed")
	UnknownKeyErr           = fmt.Errorf("unknown key")
)

func runeToKeyCode(r rune) (KeyCode, error) {
	if unicode.ToLower(r) != r {
		return 0, fmt.Errorf("key %q is invalid: %w", string(r), OnlyLowerCaseAllowedErr)
	}
	keyString := fmt.Sprintf("KEY_%s", string(unicode.ToUpper(r)))
	key, ok := evdev.KEYFromString[keyString]
	if !ok {
		return 0, fmt.Errorf("failed to get key %q: %w", keyString, UnknownKeyErr)
	}
	return key, nil
}
