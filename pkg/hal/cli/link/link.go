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
	create := NewCmdCreate(cmdutil.CommandName(commandName, parent))
	l := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", commandName),
		Short:   "Manage links",
		Long:    `Manage links`,
		Args:    cobra.NoArgs,
		Example: fmt.Sprintf("%s", create.Example),
	}

	l.AddCommand(create)

	return l
}
