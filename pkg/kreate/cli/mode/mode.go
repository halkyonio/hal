package mode

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/snowdrop/component-operator/pkg/apis/component/v1alpha2"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

const commandName = "mode"

var knownModes = map[string]bool{v1alpha2.DevDeploymentMode.String(): true, v1alpha2.BuildDeploymentMode.String(): true}
var knownModesAsString = getKnownModesAsString()

type options struct {
	mode string
	*cmdutil.ComponentTargetingOptions
}

func getKnownModesAsString() string {
	modes := make([]string, 0, len(knownModes))
	for mode := range knownModes {
		modes = append(modes, mode)
	}
	return strings.Join(modes, ",")
}

func validate(mode string) error {
	if !knownModes[mode] {
		return fmt.Errorf("unknown mode: %s, valid modes are: %s", mode, knownModesAsString)
	}
	return nil
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *options) Validate() error {
	return validate(o.mode)
}

func (o *options) Run() error {
	client := k8s.GetClient()
	patch := fmt.Sprintf(`{"spec":{"deploymentMode":"%s"}}`, o.mode)

	component, err := client.DevexpClient.Components(client.Namespace).
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
	o := &options{}
	mode := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", commandName),
		Short:   "Switch the component to the provided mode",
		Long:    `Switch the component to the provided mode.`,
		Aliases: []string{"switch"},
		Args:    cobra.NoArgs,
	}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(o, mode)
	mode.Flags().StringVarP(&o.mode, "mode", "m", "", "Mode to switch to. Possible values: "+knownModesAsString)
	return mode
}
