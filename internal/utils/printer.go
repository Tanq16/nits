package utils

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
)

var (
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa")) // blue
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")) // green
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8")) // red
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")) // yellow
)

// PrintInfo prints an info message in blue
func PrintInfo(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else {
		fmt.Println(infoStyle.Render("→ " + msg))
	}
}

// PrintSuccess prints a success message in green
func PrintSuccess(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else {
		fmt.Println(successStyle.Render("✓ " + msg))
	}
}

// PrintError prints an error message in red (does not exit)
// When debug is enabled, also logs the actual error
func PrintError(msg string, err error) {
	if GlobalDebugFlag && err != nil {
		log.Error().Err(err).Msg(msg)
	} else {
		fmt.Println(errorStyle.Render("✗ " + msg))
	}
}

// PrintFatal prints an error message and exits
// When debug is enabled, also logs the actual error
func PrintFatal(msg string, err error) {
	if GlobalDebugFlag && err != nil {
		log.Error().Err(err).Msg(msg)
	} else {
		fmt.Println(errorStyle.Render("✗ " + msg))
	}
	os.Exit(1)
}

// PrintWarn prints a warning message in yellow
// When debug is enabled, also logs the actual error
func PrintWarn(msg string, err error) {
	if GlobalDebugFlag && err != nil {
		log.Warn().Err(err).Msg(msg)
	} else {
		fmt.Println(warnStyle.Render("! " + msg))
	}
}

// PrintGeneric prints plain text without styling
func PrintGeneric(msg string) {
	fmt.Println(msg)
}
