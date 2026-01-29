package cmd

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/utils"
)

var AppVersion = "dev-build" // Set at build time via ldflags
var debugFlag bool

var rootCmd = &cobra.Command{
	Use:     "nits",
	Short:   "A collection of tiny tools and scripts",
	Version: AppVersion,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func setupLogs() {
	zerolog.TimeFieldFormat = time.DateTime
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.DateTime})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debugFlag {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		utils.GlobalDebugFlag = true
	}
}

func init() {
	// Hide default help and completion commands
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Global debug flag
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")

	// Initialize logging on startup
	cobra.OnInitialize(setupLogs)
}
