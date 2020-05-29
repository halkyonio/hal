package component

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/hal/cli/capability"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
)

const bindCommandName = "bind"

type bindOptions struct {
	component *v1beta1.Component
	*cmdutil.ComponentTargetingOptions
	capability string
}

func (o *bindOptions) SetTargetingOptions(options *cmdutil.ComponentTargetingOptions) {
	o.ComponentTargetingOptions = options
}

func (o *bindOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	// get the targeted component
	o.component, err = Entity.GetTyped(o.GetTargetedComponentName())
	if err != nil {
		return err
	}

	// get list of required capabilities and check if they are already bound
	for i, required := range o.component.Spec.Capabilities.Requires {
		// filter capabilities that don't match the requirements
		matching := capability.Entity.GetMatching(required.Spec)

		// only consider unbound capabilities for now
		isBound := len(required.BoundTo) > 0
		if isBound {
			specified, found := matching.GetByName(required.BoundTo)
			if found {
				ui.OutputSelection("Already bound capability", specified.Display())
			} else {
				ui.OutputError(fmt.Sprintf("No capability matching %v named %s was found", required.Spec, required.BoundTo))
			}
		}
		if !isBound || ui.Proceed("Change bound capability") {
			// ask user to select which matching capability to bind
			selected := ui.SelectDisplayable("Matching capability", matching)
			updated := required.DeepCopy()
			o.capability = selected.Name()
			updated.BoundTo = o.capability
			o.component.Spec.Capabilities.Requires[i] = *updated
		}
	}

	return nil
}

func (o *bindOptions) Validate() error {
	return nil
}

func (o *bindOptions) Run() error {
	_, err := Entity.client.Update(o.component)
	if err == nil {
		log.Successf("Successfully bound '%s' capability to '%s' component", o.capability, o.component.Name)
	}
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
