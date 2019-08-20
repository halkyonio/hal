package mode

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/snowdrop/kreate/pkg/validation"
	"github.com/spf13/cobra"
	component "halkyon.io/api/component/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

const commandName = "mode"

type options struct {
	mode validation.EnumValue
	*cmdutil.ComponentTargetingOptions
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *options) Validate() error {
	return o.mode.Contains(o.mode)
}

func (o *options) Run() error {
	client := k8s.GetClient()
	patch := fmt.Sprintf(`{"spec":{"deploymentMode":"%s"}}`, o.mode)

	component, err := client.HalkyonComponentClient.Components(client.Namespace).
		Patch(o.ComponentName, types.MergePatchType, []byte(patch))
	if err != nil {
		return err
	}

	logrus.Info("Component " + component.Name + " switched to " + component.Spec.DeploymentMode.String())
	return nil
}

func (o *options) SetTargetingOptions(options *cmdutil.ComponentTargetingOptions) {
	o.ComponentTargetingOptions = options
}

func NewCmdMode(parent string) *cobra.Command {
	o := &options{
		mode: validation.NewEnumValue("mode", component.DevDeploymentMode, component.BuildDeploymentMode),
	}
	mode := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", commandName),
		Short:   "Switch the component to the provided mode",
		Long:    `Switch the component to the provided mode.`,
		Aliases: []string{"switch"},
		Args:    cobra.NoArgs,
	}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(o, mode)
	mode.Flags().StringVarP(&o.mode.Provided, "mode", "m", "", "Mode to switch to. Possible values: "+o.mode.GetKnownValues())
	return mode
}
