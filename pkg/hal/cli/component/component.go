package component

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
)

const commandName = "component"

func NewCmdComponent(parent string) *cobra.Command {
	fullName := cmdutil.CommandName(commandName, parent)
	project := NewCmdProject(fullName)
	push := NewCmdPush(fullName)
	mode := NewCmdMode(fullName)
	create := NewCmdCreate(fullName)
	del := NewCmdDelete(fullName)

	hal := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Manage components",
		Long:  `Manage components`,
		Example: fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s",
			create.Example,
			del.Example,
			project.Example,
			push.Example,
			mode.Example),
	}

	hal.AddCommand(
		create,
		del,
		project,
		push,
		mode,
	)

	return hal
}
