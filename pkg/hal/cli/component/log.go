package component

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
)

const logCommandName = "log"

type logOptions struct {
	component *v1beta1.Component
	*cmdutil.ComponentTargetingOptions
}

func (o *logOptions) SetTargetingOptions(options *cmdutil.ComponentTargetingOptions) {
	o.ComponentTargetingOptions = options
}

func (o *logOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	// get the targeted component
	o.component, err = Entity.GetTyped(o.GetTargetedComponentName())
	if err != nil {
		return err
	}

	return nil
}

func (o *logOptions) Validate() error {
	return nil
}

func (o *logOptions) Run() error {
	podName := o.component.Status.GetAssociatedPodName()
	if err := k8s.Logs(podName); err != nil {
		return err
	}

	return nil
}

func NewCmdLog(fullParentName string) *cobra.Command {
	o := &logOptions{}
	bind := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", logCommandName),
		Short:   "Retrieve the logs for the component",
		Long:    `Retrieve the logs for the component.`,
		Example: fmt.Sprintf(modeExample, cmdutil.CommandName(logCommandName, fullParentName)),
		Args:    cobra.NoArgs,
	}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(o, bind)
	return bind
}
