package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/generics"
	u "github.com/tanq16/nits/utils"
)

var uuidFlags struct {
	short bool
	v4    bool
}

var uuidCmd = &cobra.Command{
	Use:   "uuid",
	Short: "Generate a UUID v7, or v4 with --v4",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var str string
		var err error
		if uuidFlags.short {
			str, err = generics.GenerateShortUUIDString(uuidFlags.v4)
		} else {
			str, err = generics.GenerateUUIDString(uuidFlags.v4)
		}
		if err != nil {
			u.PrintFatal("Failed to generate UUID", err)
		}
		u.PrintGeneric(str)
	},
}

func init() {
	uuidCmd.Flags().BoolVar(&uuidFlags.short, "short", false, "Generate a short UUID of length 18, stripping non-random bits")
	uuidCmd.Flags().BoolVar(&uuidFlags.v4, "v4", false, "Generate a UUID v4 instead of v7")
}
