package initialize

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/spf13/cobra"
	"os/exec"
	"path/filepath"
)

const commandName = "init"

type options struct {
	*cmdutil.TargetingOptions
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	return o.TargetingOptions.Complete(name, cmd, args)
}

func (o *options) Validate() error {
	return o.TargetingOptions.Validate()
}

func (o *options) Run() error {
	component := filepath.Join(o.Target, "target", "classes", "META-INF", "ap4k", "component.yml")
	command := exec.Command("kubectl", "apply", "-f", component, "-n", k8s.GetClient().Namespace)
	err := command.Run()
	if err != nil {
		return err
	}

	app := filepath.Base(o.Target)
	logrus.Info("Component for " + app + " initialized. Wait a few seconds for it to be ready!")
	return nil
}

func NewCmdInit(parent string) *cobra.Command {
	o := &options{
		TargetingOptions: cmdutil.NewTargetingOptions(),
	}

	init := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", commandName),
		Short:   "Initialize the component on the remote cluster",
		Long:    `Initialize the component on the remote cluster.`,
		Aliases: []string{"initialize"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.GenericRun(o, cmd, args)
		},
	}

	o.AttachFlagTo(init)

	return init
}
