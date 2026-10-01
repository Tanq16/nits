package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/network"
	u "github.com/tanq16/nits/utils"
)

var httpServerFlags struct {
	listenAddress string
	enableUpload  bool
}

var httpServerCmd = &cobra.Command{
	Use:   "http-server",
	Short: "Start a simple HTTP file server with optional file uploads",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		server := network.NewHTTPServer(&network.HTTPServerOptions{
			ListenAddress: httpServerFlags.listenAddress,
			EnableUpload:  httpServerFlags.enableUpload,
			OnRequest: func(remote, method, path string) {
				u.PrintStream(fmt.Sprintf("%s %s %s", remote, method, path))
			},
			OnInfo:  u.PrintInfo,
			OnError: u.PrintError,
		})
		if err := server.Setup(); err != nil {
			u.PrintFatal("failed to set up HTTP server", err)
		}
		u.PrintInfo(fmt.Sprintf("HTTP server started on http://%s/", httpServerFlags.listenAddress))
		if err := server.Run(); err != nil {
			u.PrintFatal("HTTP server error", err)
		}
	},
}

func init() {
	httpServerCmd.Flags().StringVarP(&httpServerFlags.listenAddress, "listen", "l", "0.0.0.0:8080", "Address and port to listen on")
	httpServerCmd.Flags().BoolVar(&httpServerFlags.enableUpload, "upload", false, "Enable file uploads via PUT requests")
}
