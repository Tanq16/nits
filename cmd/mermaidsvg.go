package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/mermaidsvg"
)

var mermaidSvgCmd = &cobra.Command{
	Use:   "mermaid-svg",
	Short: "Start web interface for Mermaid diagram to SVG/PNG conversion",
	Long:  `Starts a web server that provides an interactive interface to create Mermaid diagrams and export them as SVG files.`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		addr := ":" + port

		log.Info().Str("addr", "http://localhost"+addr).Msg("Starting Mermaid SVG server")

		if err := mermaidsvg.Run(addr); err != nil {
			log.Error().Err(err).Msg("Server failed")
			os.Exit(1)
		}
	},
}

func init() {
	mermaidSvgCmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	rootCmd.AddCommand(mermaidSvgCmd)
}
