package link

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
)

const (
	commandName = "link"
)

func NewCmdLink(parent string) *cobra.Command {
	fullName := cmdutil.CommandName(commandName, parent)
	create := NewCmdCreate(fullName)
	del := NewCmdDelete(fullName)
	l := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Manage links",
		Long:  `Manage links`,
		Args:  cobra.NoArgs,
		Example: fmt.Sprintf("%s\n\n%s",
			create.Example, del.Example),
	}

	l.AddCommand(
		create,
		del,
	)

	return l
}
