package mode

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/snowdrop/component-operator/pkg/apis/component/v1alpha2"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/spf13/cobra"
	"os/exec"
	"strings"
)

const commandName = "mode"

var knownModes = map[string]bool{string(v1alpha2.Dev): true, string(v1alpha2.Build): true}
var knownModesAsString = getKnownModesAsString()

type options struct {
	mode string
	*cmdutil.TargetingOptions
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

	// todo: fix
	/*err := client.KubeClient.CoreV1().RESTClient().
		Patch(types.MergePatchType).
		Namespace(client.Namespace).
		Resource("components").
		Name(o.TargetName).
		Body(patch).
		Do().
		Error()
	if err != nil {
		return err
	}*/

	command := exec.Command("kubectl", "patch", "cp", o.TargetName, "-p", patch, "--type=merge", "-n", client.Namespace)
	err := command.Run()
	if err != nil {
		return err
	}

	logrus.Info("Component for " + o.TargetName + " switched to " + o.mode)
	return nil
}

func (o *options) SetTargetingOptions(options *cmdutil.TargetingOptions) {
	o.TargetingOptions = options
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
