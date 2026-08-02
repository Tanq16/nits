package genericsCmd

import (
	"github.com/spf13/cobra"

	"github.com/tanq16/nits/internal/generics"
	u "github.com/tanq16/nits/utils"
)

var ConvertCmd = &cobra.Command{
	Use:     "convert [converter] [data or file]",
	Aliases: []string{"c"},
	Short:   "Convert data between different formats and encodings",
	Long: `Convert data between different formats and encodings.

Examples:
  nits convert docker-compose "docker run ..."  # Convert docker run to compose
  nits convert compose-docker compose.yaml      # Convert compose to docker run
  nits convert url "Hello World"                # URL encode text
  nits convert urld "Hello%20World"             # URL decode text
  nits convert jwtd "$TOKEN"                    # Decode JWT token`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		converterType := args[0]
		input := args[1]
		if err := generics.ConvertData(converterType, input); err != nil {
			u.PrintFatal("conversion failed", err)
		}
	},
}
