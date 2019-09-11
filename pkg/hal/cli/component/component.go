package component

import (
	"fmt"
	"github.com/spf13/cobra"
)

const commandName = "component"



func NewCmdComponent(parent string) *cobra.Command {
    project := NewCmdProject(commandName)
    push := NewCmdPush(commandName)
    mode := NewCmdMode(commandName)


	hal := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Manage components",
		Long:  `Manage components`,
		Example: fmt.Sprintf("%s\n\n%s\n\n%s",
        			project.Example,
        			push.Example,
        			mode.Example),
	}

	hal.AddCommand(
		project,
		push,
		mode,
	)

	return hal
}
