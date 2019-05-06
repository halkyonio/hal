package push

import (
	"bufio"
	"fmt"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/spf13/cobra"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
	"path/filepath"
)

const commandName = "push"

type options struct {
	*cmdutil.TargetingOptions
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *options) Validate() error {
	return nil
}

func (o *options) Run() error {
	c := k8s.GetClient()
	pods, err := c.KubeClient.CoreV1().Pods(c.Namespace).List(metav1.ListOptions{
		LabelSelector: "app=" + o.TargetName,
		Limit:         1,
	})
	if err != nil {
		return err
	}

	podName := pods.Items[0].Name

	/*// todo: fix copy function
	err = c.CopyFile(".", podName, "/deployments", []string{"target/" + app + "-0.0.1-SNAPSHOT.jar"}, nil)
	if err != nil {
		return err
	}*/

	jar := filepath.Join(o.TargetPath, "target", o.TargetName+"-0.0.1-SNAPSHOT.jar")
	command := exec.Command("kubectl", "cp", jar, fmt.Sprintf("%s:/deployments/app.jar", podName), "-n", c.Namespace)
	err = command.Run()
	if err != nil {
		return err
	}

	// use pipes to write output from ExecCMDInContainer in yellow  to 'out' io.Writer
	pipeReader, pipeWriter := io.Pipe()
	var cmdOutput string

	// This Go routine will automatically pipe the output from ExecCMDInContainer to
	// our logger.
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			line := scanner.Text()
			cmdOutput += fmt.Sprintln(line)
		}
	}()

	err = c.ExecCMDInContainer(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "stop", "run-java"}, pipeWriter, pipeWriter, nil, false)
	if err != nil {
		return err
	}

	err = c.ExecCMDInContainer(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "start", "run-java"}, pipeWriter, pipeWriter, nil, false)
	if err != nil {
		return err
	}

	return nil
}

func (o *options) SetTargetingOptions(options *cmdutil.TargetingOptions) {
	o.TargetingOptions = options
}

func NewCmdPush(parent string) *cobra.Command {
	push := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Push a local project to the remote cluster you're connected to",
		Long:  `Push a local project to the remote cluster you're connected to.`,
		Args:  cobra.NoArgs,
	}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(&options{}, push)
	return push
}
