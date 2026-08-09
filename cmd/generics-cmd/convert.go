package genericsCmd

import (
	"fmt"

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
		res, err := generics.ConvertData(converterType, input)
		if err != nil {
			u.PrintFatal("conversion failed", err)
		}
		if res.OutputFile != "" {
			u.PrintSuccess(fmt.Sprintf("Docker run command converted to Docker Compose: %s", res.OutputFile))
		}
		if len(res.Commands) > 0 {
			u.PrintInfo("Docker run commands for services in Docker Compose file:")
			for _, cmdStr := range res.Commands {
				u.PrintSuccess(cmdStr)
			}
		}
		if res.Output != "" {
			u.PrintGeneric(res.Output)
		}
		for _, table := range res.Tables {
			u.PrintTable(table.Headers, table.Rows)
		}
	},
}

