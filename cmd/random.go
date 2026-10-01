package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/generics"
	u "github.com/tanq16/nits/utils"
)

var randomFlags struct {
	length int
	hex    bool
	digits bool
	alpha  bool
	all    bool
}

var randomCmd = &cobra.Command{
	Use:     "random-string",
	Aliases: []string{"random"},
	Short:   "Generate a random alphanumeric string",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		charset := generics.CharsetAlphaNum
		switch {
		case randomFlags.hex:
			charset = generics.CharsetHex
		case randomFlags.digits:
			charset = generics.CharsetDigits
		case randomFlags.alpha:
			charset = generics.CharsetAlpha
		case randomFlags.all:
			charset = generics.CharsetAll
		}
		str, err := generics.GenerateRandomStringCharset(randomFlags.length, charset)
		if err != nil {
			u.PrintFatal("Failed to generate random string", err)
		}
		u.PrintGeneric(str)
	},
}

func init() {
	randomCmd.Flags().IntVarP(&randomFlags.length, "length", "l", 48, "Length of random string")
	randomCmd.Flags().BoolVar(&randomFlags.hex, "hex", false, "Use hexadecimal characters only")
	randomCmd.Flags().BoolVar(&randomFlags.digits, "digits", false, "Use digits only")
	randomCmd.Flags().BoolVar(&randomFlags.alpha, "alpha", false, "Use letters only")
	randomCmd.Flags().BoolVar(&randomFlags.all, "all", false, "Use letters, digits, and special characters")
	randomCmd.MarkFlagsMutuallyExclusive("hex", "digits", "alpha", "all")
}
