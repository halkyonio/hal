package component

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
)

const commandName = "component"

func NewCmdComponent(parent string) *cobra.Command {
	fullName := cmdutil.CommandName(commandName, parent)
	push := NewCmdPush(fullName)
	mode := NewCmdMode(fullName)
	create := NewCmdCreate(fullName)
	del := NewCmdDelete(fullName)
	bind := NewCmdBind(fullName)

	hal := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Manage components",
		Long:  `Manage components`,
		Example: fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
			create.Example,
			del.Example,
			push.Example,
			mode.Example),
	}

	hal.AddCommand(
		create,
		del,
		push,
		mode,
		bind,
		NewCmdLog(fullName),
	)

	return hal
}
