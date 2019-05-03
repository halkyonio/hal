package initialize

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

const commandName = "init"

type options struct {
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *options) Validate() error {
	return nil
}

func (o *options) Run() error {
	component := filepath.Join("target", "classes", "META-INF", "ap4k", "component.yml")
	command := exec.Command("kubectl", "apply", "-f", component, "-n", k8s.GetClient().Namespace)
	err := command.Run()
	if err != nil {
		return err
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	app := filepath.Base(currentDir)
	logrus.Info("Component for " + app + " initialized. Wait a few seconds for it to be ready!")
	return nil
}

func NewCmdInit(parent string) *cobra.Command {
	o := &options{}

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

	return init
}
