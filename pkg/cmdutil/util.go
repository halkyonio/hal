package cmdutil

import "github.com/spf13/cobra"

func CommandName(name, fullParentName string) string {
	return fullParentName + " " + name
}

// FlagValueIfSet retrieves the value of the specified flag if it is set for the given command
func FlagValueIfSet(cmd *cobra.Command, flagName string) string {
	flag, _ := cmd.Flags().GetString(flagName)
	return flag
}

func IsInteractive(cmd *cobra.Command) bool {
	return cmd.Flags().NFlag() <= 2 // heuristics to determine whether we're running in interactive mode
}
