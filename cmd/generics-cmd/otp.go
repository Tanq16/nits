package genericsCmd

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/otp"
	u "github.com/tanq16/nits/utils"
)

var OTPCmd = &cobra.Command{
	Use:   "otp",
	Short: "Store TOTP secrets locally and generate current codes",
}

var otpAddCmd = &cobra.Command{
	Use:   "add <name> <secret|otpauth-uri>",
	Short: "Store a base32 secret or an otpauth://totp/ URI under a name",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := otp.Add(args[0], args[1]); err != nil {
			u.PrintFatal("failed to add "+args[0], err)
		}
		u.PrintSuccess("added " + args[0])
	},
}

var otpGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Print the current code for a name",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		code, err := otp.Get(args[0])
		if err != nil {
			u.PrintFatal("failed to get "+args[0], err)
		}
		u.PrintGeneric(code)
	},
}

var otpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored names",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		names, entries, err := otp.List()
		if err != nil {
			u.PrintFatal("failed to list secrets", err)
		}
		if len(names) == 0 {
			u.PrintInfo("No secrets stored")
			return
		}
		rows := make([][]string, 0, len(names))
		for _, name := range names {
			e := entries[name]
			rows = append(rows, []string{name, e.Algorithm, strconv.Itoa(e.Digits), strconv.Itoa(e.Period) + "s"})
		}
		u.PrintTable([]string{"Name", "Algorithm", "Digits", "Period"}, rows)
	},
}

var otpDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a stored secret",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := otp.Delete(args[0]); err != nil {
			u.PrintFatal("failed to delete "+args[0], err)
		}
		u.PrintSuccess("deleted " + args[0])
	},
}

func init() {
	OTPCmd.AddCommand(otpAddCmd)
	OTPCmd.AddCommand(otpGetCmd)
	OTPCmd.AddCommand(otpListCmd)
	OTPCmd.AddCommand(otpDeleteCmd)
}
