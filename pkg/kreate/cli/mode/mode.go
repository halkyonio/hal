package mode

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/spf13/cobra"
	"os/exec"
)

const commandName = "mode"

type options struct {
	mode string
	*cmdutil.TargetingOptions
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *options) Validate() error {
	return nil
}

func (o *options) Run() error {
	client := k8s.GetClient()
	var mode string
	switch o.mode {
	case "dev":
		mode = "innerloop"
	case "prod":
		mode = "outerloop"
	default:
		return fmt.Errorf("unknown mode: %s, valid modes are: dev,prod", o.mode)
	}

	patch := fmt.Sprintf(`{"spec":{"deploymentMode":"%s"}}`, mode)

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
	mode.Flags().StringVarP(&o.mode, "mode", "m", "", "Mode ('dev' or 'prod') to switch to")
	return mode
}
