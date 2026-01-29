package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/mermaidsvg"
	"github.com/tanq16/nits/internal/utils"
)

var mermaidSvgCmd = &cobra.Command{
	Use:   "mermaid-svg",
	Short: "Start web interface for Mermaid diagram to SVG/PNG conversion",
	Long:  `Starts a web server that provides an interactive interface to create Mermaid diagrams and export them as SVG files.`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		addr := ":" + port

		utils.PrintInfo(fmt.Sprintf("Starting Mermaid SVG server on http://localhost%s", addr))
		log.Debug().Str("package", "cmd").Str("addr", "http://localhost"+addr).Msg("Starting server")

		if err := mermaidsvg.Run(addr); err != nil {
			utils.PrintFatal("Server failed", err)
		}
	},
}

func init() {
	mermaidSvgCmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	rootCmd.AddCommand(mermaidSvgCmd)
}
