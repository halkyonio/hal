package component

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"github.com/mholt/archiver"
	"github.com/spf13/cobra"
	component "halkyon.io/api/component/v1beta1"
	"halkyon.io/api/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/log"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"os"
	"path/filepath"
	"strings"
)

const pushCommandName = "push"

type pushOptions struct {
	*cmdutil.ComponentTargetingOptions
	binary bool
}

func (o *pushOptions) SetTargetingOptions(options *cmdutil.ComponentTargetingOptions) {
	o.ComponentTargetingOptions = options
}

var (
	pushExample = ktemplates.Examples(`  # Deploy the components client-sb, backend-sb
  %[1]s -c client-sb,backend-sb`)
	excludedFileNames = map[string]bool{
		"target": true,
	}
)

func (o *pushOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *pushOptions) Validate() (err error) {
	return nil
}

func (o *pushOptions) Run() error {
	// first check that the component exists:
	name := o.GetTargetedComponentName()
	comp, err := Entity.GetTyped(name)
	if err != nil {
		// check error to see if it means that the component doesn't exist yet
		if errors.IsNotFound(err) {
			return fmt.Errorf("no component named '%s' exists, please create it first", name)
		} else {
			return err
		}
	}

	// check if the component revision is different
	binaryPath, err := o.getComponentBinaryPath()
	if err != nil {
		return fmt.Errorf("couldn't find binary to push: %s", binaryPath)
	}
	if !o.binary {
		// first check execution context: if we're executing in the component's directory, we don't need to prepend it to the file
		currentDir, err := os.Getwd()
		if err != nil {
			return err
		}
		isInComponentDir := filepath.Base(currentDir) == o.GetTargetedComponentName()

		// generate tar
		children, err := ioutil.ReadDir(o.GetTargetedComponentPath())
		toTar := make([]string, 0, len(children))
		if err != nil {
			return err
		}
		for _, child := range children {
			name := child.Name()
			if !strings.HasPrefix(name, ".") && !excludedFileNames[name] {
				fileName := name
				if !isInComponentDir {
					fileName = filepath.Join(o.GetTargetedComponentName(), name)
				}
				toTar = append(toTar, fileName)
			}
		}

		// create tar file
		tar := archiver.NewTar()
		tar.OverwriteExisting = true
		if err := tar.Archive(toTar, binaryPath); err != nil {
			return err
		}
		defer os.Remove(binaryPath)
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
	pushType := "source code"
	if o.binary {
		pushType = "packaged binary"
	}
	log.Infof("Local changes detected for '%s' component: about to push %s to remote cluster", name, pushType)

	err = o.push(comp)
	if err != nil {
		return err
	}

	// update the component revision
	patch := fmt.Sprintf(`{"spec":{"revision":"%s"}}`, revision)
	comp.Spec.Revision = revision
	_, err = Entity.client.Patch(name, types.MergePatchType, []byte(patch))
	if err != nil {
		return err
	}

	/// create or update halkyon descriptor
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	// if we're already in the component's dir, use that, otherwise use child directory if we're dealing with a component
	componentDir := currentDir
	if filepath.Base(currentDir) != comp.Name {
		componentDir = filepath.Join(currentDir, comp.Name)
	}
	// remove Status
	comp.Status = component.ComponentStatus{}
	comp.TypeMeta = typeMeta()
	err = cmdutil.CreateOrUpdateHalkyonDescriptorWith(comp, componentDir)
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

	podName := c.Status.GetAssociatedPodName()
	// todo: review if we still need to call IsJarPresent (and if logic needs to change)
	return len(podName) > 0 && !k8s.IsJarPresent(podName)
}

func (o *pushOptions) push(component *component.Component) error {
	// wait for component to be ready
	cp, err := o.waitUntilReady(component)
	if err != nil {
		return err
	}

	c := k8s.GetClient()
	podName := cp.Status.GetAssociatedPodName()
	toPush, _ := o.getComponentBinaryPath()
	s := log.Spinner("Uploading " + toPush)
	defer s.End(false)
	err = k8s.Copy(toPush, c.Namespace, podName, !o.binary)
	if err != nil {
		return fmt.Errorf("error uploading file: %v", err)
	}
	s.End(true)

	if !o.binary {
		// clean up any existing code to avoid getting remnants from all code
		if err = c.ExecCommand(podName, []string{"rm", "-rf", k8s.ExtractedSourcePathInContainer + "/*"},
			"Cleaning up component"); err != nil {
			return err
		}

		if err = c.ExecCommand(podName, []string{"tar", "xmf", k8s.SourcePathInContainer, "-C", k8s.ExtractedSourcePathInContainer},
			"Extracting source on the remote cluster"); err != nil {
			return err
		}

		if err = c.ExecCommand(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "start", "build"}, "Performing build"); err != nil {
			return err
		}

		if err = c.ExecCommand(podName, []string{"bash", "-c", "while /var/lib/supervisord/bin/supervisord ctl status build | grep RUNNING; do sleep 1; done"},
			"Waiting for build to finish"); err != nil {
			return err
		}
	}

	if err = c.ExecCommand(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "stop", "run"}, ""); err != nil {
		return err
	}
	if err = c.ExecCommand(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "start", "run"}, "Restarting app"); err != nil {
		return err
	}
	log.Successf("Successfully pushed '%s' component to remote cluster", component.Name)
	return nil
}

func (o *pushOptions) getComponentBinaryPath() (string, error) {
	if !o.binary {
		currentDir, _ := os.Getwd()
		return filepath.Join(currentDir, o.GetTargetedComponentName()+".tar"), nil
	}

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

func (o *pushOptions) waitUntilReady(c *component.Component) (*component.Component, error) {
	if v1beta1.ReasonReady == c.Status.Reason {
		return c, nil
	}

	name := o.GetTargetedComponentName()
	client := k8s.GetClient()
	cp, err := client.WaitForComponent(name, component.PushReady, "Waiting for component "+name+" to be ready…")
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
	switch c.Status.Reason {
	case v1beta1.ReasonFailed:
		return fmt.Errorf("status of component %s is %s: %s", c.Name, c.Status.Reason, c.Status.Message)
	default:
		return nil
	}
}

func NewCmdPush(fullParentName string) *cobra.Command {
	push := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", pushCommandName),
		Short:   "Push a local project to the remote cluster you're connected to",
		Long:    `Push a local project to the remote cluster you're connected to.`,
		Example: fmt.Sprintf(pushExample, cmdutil.CommandName(pushCommandName, fullParentName)),
		Args:    cobra.NoArgs,
	}
	options := pushOptions{}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(&options, push)
	push.Flags().BoolVarP(&options.binary, "binary", "b", false, "Push packaged binary instead of source code")
	return push
}
