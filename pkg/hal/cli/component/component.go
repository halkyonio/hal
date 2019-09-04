package component

import (
	"fmt"
	"github.com/spf13/cobra"
)

const commandName = "component"

func NewCmdComponent(parent string) *cobra.Command {
	hal := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Manage components",
		Long:  `Manage components`,
	}

	hal.AddCommand(
		NewCmdProject(commandName),
		NewCmdPush(commandName),
		NewCmdMode(commandName),
	)

	return hal
}
