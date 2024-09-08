package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/holoplot/go-evdev"
	"gopkg.in/yaml.v3"
)

type Yaml struct {
	Combos []struct {
		Keys    string `yaml:"keys"`
		OutKeys string `yaml:"outKeys"`
	} `yaml:"combos"`
}

func LoadYamlFile(yamlFile string) ([]*Combo, error) {
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml config from %q: %w", yamlFile, err)
	}
	combos, err := LoadYamlFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", yamlFile, err)
	}
	return combos, nil
}

func LoadYamlFromBytes(yamlBytes []byte) ([]*Combo, error) {
	y := Yaml{}
	err := yaml.Unmarshal(yamlBytes, &y)
	if err != nil {
		return nil, err
	}
	combos := make([]*Combo, 0, len(y.Combos))
	for _, yamlCombo := range y.Combos {
		combo := Combo{}
		if len(yamlCombo.Keys) == 0 {
			return nil, fmt.Errorf("empty list in 'keys' is not allowed.")
		}
		keys, err := stringToKeyCodes(yamlCombo.Keys)
		if err != nil {
			return nil, err
		}
		combo.Keys = keys

		if len(yamlCombo.OutKeys) == 0 {
			return nil, fmt.Errorf("empty list in 'outKeys' is not allowed.")
		}
		keys, err = stringToKeyCodes(yamlCombo.OutKeys)
		if err != nil {
			return nil, err
		}
		combo.OutKeys = keys
		combos = append(combos, &combo)
	}
	return combos, nil
}

func stringToKeyCodes(str string) ([]KeyCode, error) {
	words := strings.Fields(str)
	codes := make([]KeyCode, len(words))
	for i, word := range words {
		keyCode, err := wordToKeyCode(word)
		if err != nil {
			return nil, err
		}
		codes[i] = keyCode
	}
	return codes, nil
}

// asdfasdfasdf

var (
	OnlyLowerCaseAllowedErr = fmt.Errorf("only lower case characters are allowed")
	UnknownKeyErr           = fmt.Errorf("unknown key")
)

func wordToKeyCode(s string) (KeyCode, error) {
	if strings.ToLower(s) != s {
		return 0, fmt.Errorf("key %q is invalid: %w", s, OnlyLowerCaseAllowedErr)
	}
	keyString := fmt.Sprintf("KEY_%s", string(strings.ToUpper(s)))
	key, ok := evdev.KEYFromString[keyString]
	if !ok {
		return 0, fmt.Errorf("failed to get key %q: %w. Use sub-command 'print' to see valid names of keys", s, UnknownKeyErr)
	}
	return key, nil
}
