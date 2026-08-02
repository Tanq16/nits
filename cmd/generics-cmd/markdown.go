package genericsCmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/generics"
	u "github.com/tanq16/nits/utils"
)

var markdownFlags struct {
	listenAddress string
}

var MarkdownCmd = &cobra.Command{
	Use:     "markdown",
	Aliases: []string{"md"},
	Short:   "Start a markdown viewer web server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := generics.StartMarkdownServer(markdownFlags.listenAddress); err != nil {
			u.PrintFatal("Failed to start markdown viewer", err)
		}
	},
}

func init() {
	MarkdownCmd.Flags().StringVarP(&markdownFlags.listenAddress, "listen", "l", "0.0.0.0:8080", "Address and port to listen on")
}
