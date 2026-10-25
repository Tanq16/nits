package cmd

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	archiveCmd "github.com/tanq16/nits/cmd/archive-cmd"
	timeCmd "github.com/tanq16/nits/cmd/time-cmd"
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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return utils.ResolveStdin(cmd)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
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

	addGroup("network", "Network:", downloadCmd, gitHubReleaseCmd, httpServerCmd, uploadCmd, ipInfoCmd)
	addGroup("files", "Files:", archiveCmd.ArchiveCmd, bulkRenameCmd, duplicatesCmd, fileUnzipperCmd, imgDedupeCmd)
	addGroup("generators", "Generators and Time:", passphraseCmd, uuidCmd, randomCmd, timeCmd.TimeCmd)
	addGroup("data", "Data:", convertCmd, neo4jCmd)
	rootCmd.SetHelpCommandGroupID("data")
}

func addGroup(id, title string, cmds ...*cobra.Command) {
	rootCmd.AddGroup(&cobra.Group{ID: id, Title: title})
	for _, c := range cmds {
		c.GroupID = id
		rootCmd.AddCommand(c)
	}
}
