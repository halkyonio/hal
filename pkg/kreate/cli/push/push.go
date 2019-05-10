package push

import (
	"bufio"
	"fmt"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/snowdrop/kreate/pkg/log"
	"github.com/spf13/cobra"
	"io"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	selector := "app=" + o.TargetName
	pods, err := c.KubeClient.CoreV1().Pods(c.Namespace).List(metav1.ListOptions{
		LabelSelector: selector,
		Limit:         1,
	})
	if err != nil {
		return err
	}

	var podName string
	if len(pods.Items) == 0 {
		// the pod doesn't exist, create it and wait for it to be ready
		component := filepath.Join(o.TargetPath, "target", "classes", "META-INF", "ap4k", "component.yml")

		err := k8s.Apply(component, c.Namespace)
		if err != nil {
			return fmt.Errorf("error applying component CR: %v", err)
		}

		pod, err := c.WaitAndGetPod(selector, v1.PodRunning, "Component for "+o.TargetName+" initialized. Waiting for it to be ready…")
		if err != nil {
			return fmt.Errorf("error waiting for pod: %v", err)
		}

		podName = pod.Name
	} else {
		podName = pods.Items[0].Name
	}

	/*// todo: fix copy function
	err = c.CopyFile(".", podName, "/deployments", []string{"target/" + app + "-0.0.1-SNAPSHOT.jar"}, nil)
	if err != nil {
		return err
	}*/

	jar := filepath.Join(o.TargetPath, "target", o.TargetName+"-0.0.1-SNAPSHOT.jar")
	s := log.Spinner("Uploading " + jar)
	defer s.End(false)
	err = k8s.Copy(jar, c.Namespace, podName)
	if err != nil {
		return fmt.Errorf("error uploading jar: %v", err)
	}
	s.End(true)

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
