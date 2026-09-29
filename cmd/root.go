package cmd

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	genericsCmd "github.com/tanq16/nits/cmd/generics-cmd"
	interactionsCmd "github.com/tanq16/nits/cmd/interactions-cmd"
	networkCmd "github.com/tanq16/nits/cmd/network-cmd"
	"github.com/tanq16/nits/utils"
)

var AppVersion = "dev-build"
var debugFlag bool

var rootCmd = &cobra.Command{
	Use:     "nits",
	Short:   "A collection of tiny tools and scripts",
	Version: AppVersion,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return utils.ResolveStdin(cmd)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		utils.PrintFatal("Command failed", err)
	}
}

func setupLogs() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	var out io.Writer = os.Stdout
	if utils.StdoutIsTerminal {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}
	}
	log.Logger = zerolog.New(out).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debugFlag {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		utils.GlobalDebugFlag = true
	}
}

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")
	cobra.OnInitialize(setupLogs)

	rootCmd.AddCommand(networkCmd.DownloadCmd)
	rootCmd.AddCommand(networkCmd.GitHubReleaseCmd)
	rootCmd.AddCommand(networkCmd.HTTPServerCmd)
	rootCmd.AddCommand(networkCmd.IPInfoCmd)

	rootCmd.AddCommand(genericsCmd.ArchiveCmd)
	rootCmd.AddCommand(genericsCmd.BulkRenameCmd)
	rootCmd.AddCommand(genericsCmd.DuplicatesCmd)
	rootCmd.AddCommand(genericsCmd.PassphraseCmd)
	rootCmd.AddCommand(genericsCmd.UUIDCmd)
	rootCmd.AddCommand(genericsCmd.RandomCmd)
	rootCmd.AddCommand(genericsCmd.TimeCmd)
	rootCmd.AddCommand(genericsCmd.ManualRenameCmd)
	rootCmd.AddCommand(genericsCmd.ConvertCmd)
	rootCmd.AddCommand(genericsCmd.MarkdownCmd)

	rootCmd.AddCommand(interactionsCmd.FSSyncCmd)
	rootCmd.AddCommand(interactionsCmd.Neo4jCmd)
}
