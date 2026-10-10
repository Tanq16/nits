package archiveCmd

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/archive"
	u "github.com/tanq16/nits/utils"
)

var archiveFlags struct {
	output   string
	include  []string
	exclude  []string
	bare     bool
	encrypt  bool
	password string
}

var archiveExtractFlags struct {
	bare     bool
	password string
}

var archiveUnrarFlags struct {
	bare     bool
	password string
}

var ArchiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Create or extract zip archives, and extract rar archives",
}

var archiveCreateCmd = &cobra.Command{
	Use:     "create <path> [path...]",
	Aliases: []string{"c"},
	Short:   "Create a zip archive from files and directories",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		include := compileRegexes(archiveFlags.include, "include")
		exclude := compileRegexes(archiveFlags.exclude, "exclude")
		password := archiveFlags.password
		encrypt := archiveFlags.encrypt || password != ""
		if encrypt && password == "" {
			password = promptPassword("archive create --encrypt needs --password, or --password -", "archive create needs a non-empty password")
		}
		output := archive.OutputPath(archiveFlags.output, encrypt)
		cfg := archive.CreateConfig{
			Paths:    args,
			Output:   output,
			Include:  include,
			Exclude:  exclude,
			Bare:     archiveFlags.bare,
			Encrypt:  encrypt,
			Password: password,
		}
		if err := archive.Create(cfg); err != nil {
			u.PrintFatal("failed to create archive", err)
		}
		u.PrintSuccess(fmt.Sprintf("created %s", output))
	},
}

var archiveExtractCmd = &cobra.Command{
	Use:     "extract <file>",
	Aliases: []string{"e"},
	Short:   "Extract a zip archive",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		encrypted, err := archive.IsEncrypted(args[0])
		if err != nil {
			u.PrintFatal("failed to read archive", err)
		}
		password := archiveExtractFlags.password
		if encrypted && password == "" {
			password = promptPassword("archive extract needs --password, or --password -", "archive extract needs a non-empty password")
		}
		cfg := archive.ExtractConfig{
			Archive:  args[0],
			Dest:     ".",
			Bare:     archiveExtractFlags.bare,
			Password: password,
		}
		if err := archive.Extract(cfg); err != nil {
			u.PrintFatal("failed to extract archive", err)
		}
		u.PrintSuccess("extracted")
	},
}

var archiveUnrarCmd = &cobra.Command{
	Use:   "unrar <file>",
	Short: "Extract a rar archive, reading later volumes of a multi-volume set automatically",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		password := archiveUnrarFlags.password
		if password == "" {
			encrypted, err := archive.IsRarEncrypted(args[0])
			if err != nil {
				u.PrintFatal("failed to read archive", err)
			}
			if encrypted {
				password = promptPassword("archive unrar needs --password, or --password -", "archive unrar needs a non-empty password")
			}
		}
		cfg := archive.ExtractConfig{
			Archive:  args[0],
			Dest:     ".",
			Bare:     archiveUnrarFlags.bare,
			Password: password,
		}
		if err := archive.Unrar(cfg); err != nil {
			u.PrintFatal("failed to extract archive", err)
		}
		u.PrintSuccess("extracted")
	},
}

func promptPassword(noTerminalMsg, emptyMsg string) string {
	entered, err := u.PromptPassword("Password:")
	if errors.Is(err, u.ErrNoTerminal) {
		u.PrintFatal(noTerminalMsg, nil)
	}
	if err != nil {
		u.PrintFatal("TUI error", err)
	}
	if entered == "" {
		u.PrintFatal(emptyMsg, nil)
	}
	return entered
}

func compileRegexes(pats []string, flagName string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(pats))
	for _, p := range pats {
		re, err := regexp.Compile(p)
		if err != nil {
			u.PrintFatal("invalid --"+flagName+" regex", err)
		}
		out = append(out, re)
	}
	return out
}

func init() {
	archiveCreateCmd.Flags().StringVarP(&archiveFlags.output, "output", "o", "archive.zip", "Output zip path")
	archiveCreateCmd.Flags().StringArrayVar(&archiveFlags.include, "include", nil, "Include only zip paths matching regex (repeatable)")
	archiveCreateCmd.Flags().StringArrayVar(&archiveFlags.exclude, "exclude", nil, "Exclude zip paths matching regex (repeatable)")
	archiveCreateCmd.Flags().BoolVar(&archiveFlags.bare, "bare", false, "Store given paths at zip root with no wrapper directory")
	archiveCreateCmd.Flags().BoolVar(&archiveFlags.encrypt, "encrypt", false, "Encrypt the zip, prompting for a password unless --password is given")
	archiveCreateCmd.Flags().StringVar(&archiveFlags.password, "password", "", "Password to encrypt the zip with (implies --encrypt), or - to read it from stdin")
	_ = u.MarkStdinLine(archiveCreateCmd, "password")

	archiveExtractCmd.Flags().BoolVar(&archiveExtractFlags.bare, "bare", false, "Strip the first path component when extracting")
	archiveExtractCmd.Flags().StringVar(&archiveExtractFlags.password, "password", "", "Password for an encrypted archive, or - to read it from stdin")
	_ = u.MarkStdinLine(archiveExtractCmd, "password")

	archiveUnrarCmd.Flags().BoolVar(&archiveUnrarFlags.bare, "bare", false, "Strip the first path component when extracting")
	archiveUnrarCmd.Flags().StringVar(&archiveUnrarFlags.password, "password", "", "Password for an encrypted archive, or - to read it from stdin")
	_ = u.MarkStdinLine(archiveUnrarCmd, "password")

	ArchiveCmd.AddCommand(archiveCreateCmd)
	ArchiveCmd.AddCommand(archiveExtractCmd)
	ArchiveCmd.AddCommand(archiveUnrarCmd)
}
