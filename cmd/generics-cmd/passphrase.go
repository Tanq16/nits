package genericsCmd

import (
	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/generics"
	u "github.com/tanq16/nits/utils"
)

var passphraseFlags struct {
	length int
	simple bool
}

var PassphraseCmd = &cobra.Command{
	Use:   "passphrase",
	Short: "Generate a passphrase",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		phrase, err := generics.GeneratePassPhrase(passphraseFlags.length, passphraseFlags.simple)
		if err != nil {
			u.PrintFatal("Failed to generate passphrase", err)
		}
		u.PrintGeneric(phrase)
	},
}

func init() {
	PassphraseCmd.Flags().IntVarP(&passphraseFlags.length, "length", "l", 3, "Number of words in passphrase")
	PassphraseCmd.Flags().BoolVar(&passphraseFlags.simple, "simple", false, "Use plain words with no capital letter or digit")
}
