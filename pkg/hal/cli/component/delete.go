package component

import (
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
)

func NewCmdDelete(fullParentName string) *cobra.Command {
	generic := cmdutil.NewDeleteOptions("component", Entity)
	return cmdutil.NewGenericDelete(fullParentName, generic)
}
