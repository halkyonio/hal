package component

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	component "halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/validation"
	"k8s.io/apimachinery/pkg/types"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const modeCommandName = "mode"

type modeOptions struct {
	mode validation.EnumValue
	*cmdutil.ComponentTargetingOptions
}

var (
	modeExample = ktemplates.Examples(`  # Switch the component backend to the provided mode
  %[1]s -c backend-sb -m dev`)
)

func (o *modeOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *modeOptions) Validate() error {
	return o.mode.Contains(o.mode)
}

func (o *modeOptions) Run() error {
	client := k8s.GetClient()
	patch := fmt.Sprintf(`{"spec":{"deploymentMode":"%s"}}`, o.mode)

	component, err := client.HalkyonComponentClient.Components(client.Namespace).
		Patch(o.GetTargetedComponentName(), types.MergePatchType, []byte(patch))
	if err != nil {
		return err
	}

	logrus.Info("Component " + component.Name + " switched to " + component.Spec.DeploymentMode.String())
	return nil
}

func (o *modeOptions) SetTargetingOptions(options *cmdutil.ComponentTargetingOptions) {
	o.ComponentTargetingOptions = options
}

func NewCmdMode(parent string) *cobra.Command {
	o := &modeOptions{
		mode: validation.NewEnumValue("mode", component.DevDeploymentMode, component.BuildDeploymentMode),
	}
	mode := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", modeCommandName),
		Short:   "Switch the component to the provided mode",
		Long:    `Switch the component to the provided mode.`,
		Example: fmt.Sprintf(modeExample, "hal component mode"),
		Aliases: []string{"switch"},
		Args:    cobra.NoArgs,
	}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(o, mode)
	mode.Flags().StringVarP(&o.mode.Provided, "mode", "m", "", "Mode to switch to. Possible values: "+o.mode.GetKnownValues())
	return mode
}
