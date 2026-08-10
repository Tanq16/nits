package generics

import (
	"fmt"
	"os"
)

type TableData struct {
	Headers []string
	Rows    [][]string
}

type ConvertResult struct {
	Output     string
	OutputFile string
	Commands   []string
	Tables     []TableData
}

type converterInfo struct {
	InputType string
	Handler   func(input string) (*ConvertResult, error)
}

var supportedConverters = map[string]converterInfo{
	"docker-compose": {
		InputType: "string",
		Handler:   convertDockerToCompose,
	},
	"compose-docker": {
		InputType: "file",
		Handler:   convertComposeToDocker,
	},
	"urld": {
		InputType: "string",
		Handler:   urlToText,
	},
	"url": {
		InputType: "string",
		Handler:   textToUrl,
	},
	"jwtd": {
		InputType: "string",
		Handler:   jwtDecode,
	},
}

func ConvertData(converterType string, input string) (*ConvertResult, error) {
	converter, exists := supportedConverters[converterType]
	if !exists {
		return nil, fmt.Errorf("unsupported converter type: %s", converterType)
	}
	switch converter.InputType {
	case "file":
		if _, err := os.Stat(input); os.IsNotExist(err) {
			return nil, fmt.Errorf("input file does not exist: %s", input)
		}
	case "string":
		if input == "" {
			return nil, fmt.Errorf("input string cannot be empty")
		}
	}
	return converter.Handler(input)
}

