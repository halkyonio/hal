package component

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/hal/cli/capability"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/ui"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const bindCommandName = "bind"

type bindOptions struct {
	component *v1beta1.Component
	*cmdutil.ComponentTargetingOptions
}

func (o *bindOptions) SetTargetingOptions(options *cmdutil.ComponentTargetingOptions) {
	o.ComponentTargetingOptions = options
}

func (o *bindOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	// get the targeted component
	client := k8s.GetClient()
	o.component, err = client.HalkyonComponentClient.Components(client.Namespace).Get(o.GetTargetedComponentName(), v1.GetOptions{})
	if err != nil {
		return err
	}

	// get list of required capabilities and check if they are already bound
	requires := o.component.Spec.Capabilities.Requires
	for i, required := range requires {
		// only consider unbound capabilities for now
		isBound := len(required.BoundTo) > 0
		if isBound {
			ui.OutputSelection("Already bound capability", required.BoundTo)
		}
		if !isBound || ui.Proceed("Change bound capability") {
			// filter capabilities that don't match the requirements
			matching := capability.Entity.GetMatching(required.Spec)

			// ask user to select which matching capability to bind
			selected := ui.Select("Matching capability", getCapabilityNames(matching))
			updated := required.DeepCopy()
			updated.BoundTo = selected
			requires[i] = *updated
		}
	}

	return nil
}

func (o *bindOptions) Validate() error {
	return nil
}

func (o *bindOptions) Run() error {
	_, err := Entity.client.Update(o.component)
	return err
}

func NewCmdBind(fullParentName string) *cobra.Command {
	o := &bindOptions{}
	bind := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", bindCommandName),
		Short:   "Bind the component to a capability",
		Long:    `Bind the component to a capability.`,
		Example: fmt.Sprintf(modeExample, cmdutil.CommandName(bindCommandName, fullParentName)),
		Args:    cobra.NoArgs,
	}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(o, bind)
	return bind
}
