package component

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/log"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const createCommandName = "create"

type createOptions struct {
	*commonOptions
}

var (
	createExample = ktemplates.Examples(`  # Create a new Halkyon component found in the 'foo' child directory of the current directory
  %[1]s -c foo`)
)

func (o *createOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *createOptions) Validate() error {
	return nil
}

func (o *createOptions) Run() error {
	comp, err := o.createIfNeeded()
	if err == nil {
		log.Successf("Successfully created '%s' component", comp.Name)
	}
	return err
}

func NewCmdCreate(fullParentName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", createCommandName),
		Short:   "Create a new Halkyon component",
		Long:    `Create a new Halkyon component by sending a dekorate-generated Halkyon descriptor to the remote cluster you're connected to'`,
		Example: fmt.Sprintf(createExample, cmdutil.CommandName(createCommandName, fullParentName)),
		Args:    cobra.NoArgs,
	}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(&createOptions{commonOptions: &commonOptions{}}, cmd)
	return cmd
}
