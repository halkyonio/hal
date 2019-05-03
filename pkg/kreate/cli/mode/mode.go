package mode

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/snowdrop/kreate/pkg/ui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

const commandName = "mode"

type options struct {
	mode string
	*cmdutil.TargetingOptions
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	return o.TargetingOptions.Complete(name, cmd, args)
}

func (o *options) Validate() error {
	return o.TargetingOptions.Validate()
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

	patch := []byte(fmt.Sprintf(`{"spec":{"deploymentMode":"%s"}}`, mode))
	err := client.KubeClient.CoreV1().RESTClient().
		Patch(types.MergePatchType).
		Namespace(client.Namespace).
		Resource("components").
		Name(o.TargetName).
		Body(patch).
		Do().
		Error()
	if err != nil {
		return err
	}

	logrus.Info("Component for " + o.TargetName + " switched to " + o.mode)
	return nil
}

func NewCmdMode(parent string) *cobra.Command {
	ui.Proceed("foo")
	o := &options{
		TargetingOptions: cmdutil.NewTargetingOptions(),
	}

	mode := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", commandName),
		Short:   "Switch the component to the provided mode",
		Long:    `Switch the component to the provided mode.`,
		Aliases: []string{"switch"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.GenericRun(o, cmd, args)
		},
	}

	mode.Flags().StringVarP(&o.mode, "mode", "m", "", "Mode ('dev' or 'prod') to switch to")
	o.AttachFlagTo(mode)

	return mode
}
