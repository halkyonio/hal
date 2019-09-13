package capability

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
)

const commandName = "capability"

func NewCmdCapability(parent string) *cobra.Command {
	fullName := cmdutil.CommandName(commandName, parent)
	create := NewCmdCreate(fullName)
	del := NewCmdDelete(fullName)

	hal := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", commandName),
		Short:   "Manage capabilities",
		Long:    `Manage capabilities`,
		Example: fmt.Sprintf("%s\n\n%s", create.Example, del.Example),
	}

	hal.AddCommand(
		create,
		del,
	)

	return hal
}
