package push

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"github.com/snowdrop/component-operator/pkg/apis/component/v1alpha2"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/snowdrop/kreate/pkg/log"
	"github.com/spf13/cobra"
	"io"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record/util"
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
	component, err := c.DevexpClient.Components(c.Namespace).Get(o.TargetName, v1.GetOptions{})
	if err != nil {
		// check error to see if it means that the component doesn't exist yet
		if util.IsKeyNotFoundError(errors.Cause(err)) {
			// the component was not found so we need to create it first and wait for it to be ready
			descriptor := filepath.Join(o.TargetPath, "target", "classes", "META-INF", "ap4k", "component.yml")

			err = k8s.Apply(descriptor, c.Namespace)
			if err != nil {
				return fmt.Errorf("error applying component CR: %v", err)
			}

			component, err = c.WaitForComponent(o.TargetName, v1alpha2.ComponentReady, "Initializing component "+o.TargetName+". Waiting for it to be ready…")
			if err != nil {
				return fmt.Errorf("error waiting for component: %v", err)
			}
		} else {
			return err
		}
	}
	podName := component.Status.PodName

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
