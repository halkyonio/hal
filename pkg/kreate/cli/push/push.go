package push

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/snowdrop/kreate/pkg/ui"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"os/exec"
	"path/filepath"
)

const commandName = "push"

type options struct {
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *options) Validate() error {
	return nil
}

func (o *options) Run() error {
	ui.Proceed("foo")
	c := k8s.GetClient()
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	app := filepath.Base(currentDir)

	pods, err := c.KubeClient.CoreV1().Pods(c.Namespace).List(metav1.ListOptions{
		LabelSelector: "app=" + app,
		Limit:         1,
	})
	if err != nil {
		return err
	}

	podName := pods.Items[0].Name

	jar := filepath.Join("target", app+"-0.0.1-SNAPSHOT.jar")

	// todo: fix copy function
	/*err = c.CopyFile(jar, podName, "/deployments")
	if err != nil {
		return err
	}*/

	command := exec.Command("kubectl", "cp", jar, fmt.Sprintf("%s:/deployments/app.jar", podName), "-n", c.Namespace)
	err = command.Run()
	if err != nil {
		return err
	}

	err = c.ExecCMDInContainer(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "stop", "run-java"}, nil, nil, nil, false)
	if err != nil {
		return err
	}

	err = c.ExecCMDInContainer(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "start", "run-java"}, nil, nil, nil, false)
	if err != nil {
		return err
	}

	return nil
}

func NewCmdPush(parent string) *cobra.Command {
	p := &options{}

	push := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Push a local project to the remote cluster you're connected to",
		Long:  `Push a local project to the remote cluster you're connected to.`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.GenericRun(p, cmd, args)
		},
	}

	return push
}
