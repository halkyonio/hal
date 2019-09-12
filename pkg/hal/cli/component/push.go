package component

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	component "halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/log"
	"io"
	"io/ioutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record/util"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"os"
	"path/filepath"
	"strings"
)

const pushCommandName = "push"

type pushOptions struct {
	*cmdutil.ComponentTargetingOptions
}

var (
	pushExample = ktemplates.Examples(`  # Deploy the components client-sb, backend-sb
  %[1]s -c client-sb,backend-sb`)
)

func (o *pushOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *pushOptions) Validate() error {
	return nil
}

func (o *pushOptions) Run() error {
	c := k8s.GetClient()
	name := o.GetTargetedComponentName()
	comp, err := c.HalkyonComponentClient.Components(c.Namespace).Get(name, v1.GetOptions{})
	if err != nil {
		// check error to see if it means that the component doesn't exist yet
		if util.IsKeyNotFoundError(errors.Cause(err)) {
			// the component was not found so we need to create it first and wait for it to be ready
			log.Infof("'%s' component was not found, initializing it", name)
			err = k8s.Apply(o.GetTargetedComponentDescriptor(), c.Namespace)
			if err != nil {
				return fmt.Errorf("error applying component CR: %v", err)
			}

			comp, err = o.waitUntilReady(comp)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// check if the component revision is different
	binaryPath, err := o.getComponentBinaryPath()
	if err != nil {
		return fmt.Errorf("couldn't find binary to push: %s", binaryPath)
	}
	file, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	input := bufio.NewReader(file)
	hash := sha1.New()
	if _, err := io.Copy(hash, input); err != nil {
		return err
	}
	revision := fmt.Sprintf("%x", hash.Sum(nil))
	if !o.needsPush(revision, comp) {
		log.Infof("No local changes detected for '%s' component: nothing to push!", name)
		return nil
	}

	// we got the component, we still need to check it's ready
	comp, err = o.waitUntilReady(comp)
	if err != nil {
		return err
	}
	err = o.push(comp)
	if err != nil {
		return err
	}

	// update the component revision
	patch := fmt.Sprintf(`{"spec":{"revision":"%s"}}`, revision)
	_, err = c.HalkyonComponentClient.Components(c.Namespace).Patch(name, types.MergePatchType, []byte(patch))
	if err != nil {
		return err
	}
	return nil

}

func (o *pushOptions) needsPush(revision string, c *component.Component) bool {
	sameRevision := revision == c.Spec.Revision
	if !sameRevision {
		return true
	}

	podName := c.Status.PodName
	return len(podName) > 0 && !k8s.IsJarPresent(podName)
}

func (o *pushOptions) waitUntilReady(c *component.Component) (*component.Component, error) {
	if component.ComponentReady == c.Status.Phase || component.ComponentRunning == c.Status.Phase {
		return c, nil
	}

	name := o.GetTargetedComponentName()
	client := k8s.GetClient()
	cp, err := client.WaitForComponent(name, component.ComponentReady, "Waiting for component "+name+" to be ready…")
	if err != nil {
		return nil, fmt.Errorf("error waiting for component: %v", err)
	}
	err = errorIfFailedOrUnknown(c)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

func errorIfFailedOrUnknown(c *component.Component) error {
	switch c.Status.Phase {
	case component.ComponentFailed, component.ComponentUnknown:
		return errors.Errorf("status of component %s is %s: %s", c.Name, c.Status.Phase, c.Status.Message)
	default:
		return nil
	}
}

func (o *pushOptions) push(component *component.Component) error {
	c := k8s.GetClient()
	podName := component.Status.PodName
	/*// todo: fix copy function
	err = c.CopyFile(".", podName, "/deployments", []string{"target/" + app + "-0.0.1-SNAPSHOT.jar"}, nil)
	if err != nil {
		return err
	}*/
	jar, _ := o.getComponentBinaryPath()
	s := log.Spinner("Uploading " + jar)
	defer s.End(false)
	err := k8s.Copy(jar, c.Namespace, podName)
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

func (o *pushOptions) getComponentBinaryPath() (string, error) {
	target := filepath.Join(o.GetTargetedComponentPath(), "target")
	files, err := ioutil.ReadDir(target)
	if err != nil {
		return target + " directory not found or unreadable", err
	}

	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, ".jar") {
			return filepath.Join(target, name), nil
		}
	}
	return "no jar file found in " + target, nil
}

func (o *pushOptions) SetTargetingOptions(options *cmdutil.ComponentTargetingOptions) {
	o.ComponentTargetingOptions = options
}

func NewCmdPush(fullParentName string) *cobra.Command {
	push := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", pushCommandName),
		Short:   "Push a local project to the remote cluster you're connected to",
		Long:    `Push a local project to the remote cluster you're connected to.`,
		Example: fmt.Sprintf(pushExample, cmdutil.CommandName(pushCommandName, fullParentName)),
		Args:    cobra.NoArgs,
	}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(&pushOptions{}, push)
	return push
}
