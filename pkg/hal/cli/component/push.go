package component

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"github.com/mholt/archiver"
	"github.com/spf13/cobra"
	component "halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/log"
	"io"
	"io/ioutil"
	k8score "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"os"
	"path/filepath"
	"strings"
)

const pushCommandName = "push"

type pushOptions struct {
	*commonOptions
	binary bool
}

var (
	pushExample = ktemplates.Examples(`  # Deploy the components client-sb, backend-sb
  %[1]s -c client-sb,backend-sb`)
	excludedFileNames = map[string]bool{
		"target":    true,
		".git":      true,
		".DS_Store": true,
		".idea":     true,
	}
)

func (o *pushOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *pushOptions) Validate() error {
	return nil
}

func (o *pushOptions) Run() error {
	comp, err := o.createIfNeeded()
	if err != nil {
		return err
	}
	name := comp.Name

	// check if the component revision is different
	binaryPath, err := o.getComponentBinaryPath()
	if err != nil {
		return fmt.Errorf("couldn't find binary to push: %s", binaryPath)
	}
	if !o.binary {
		// generate tar
		// naively exclude target from files to be tarred
		children, err := ioutil.ReadDir(o.GetTargetedComponentPath())
		toTar := make([]string, 0, len(children))
		if err != nil {
			return err
		}
		for _, child := range children {
			name := child.Name()
			if !excludedFileNames[name] {
				toTar = append(toTar, filepath.Join(o.GetTargetedComponentName(), name))
			}
		}

		// create tar file
		tar := archiver.NewTar()
		tar.OverwriteExisting = true
		if err := tar.Archive(toTar, binaryPath); err != nil {
			return err
		}
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
	c := k8s.GetClient()
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
	podName := cp.Status.PodName
	toPush, _ := o.getComponentBinaryPath()
	s := log.Spinner("Uploading " + toPush)
	defer s.End(false)
	err = k8s.Copy(toPush, c.Namespace, podName, !o.binary)
	if err != nil {
		return fmt.Errorf("error uploading file: %v", err)
	}
	s.End(true)

	if !o.binary {
		if err = c.ExecCommand(podName, []string{"tar", "xmf", k8s.SourcePathInContainer, "-C", k8s.ExtractedSourcePathInContainer},
			"Extracting source on the remote cluster"); err != nil {
			return err
		}

		// need to stop running app before building a new version
		if err = c.ExecCommand(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "stop", "run"}, ""); err != nil {
			return err
		}

		if err = c.ExecCommand(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "start", "build"}, "Performing build"); err != nil {
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

func NewCmdPush(fullParentName string) *cobra.Command {
	push := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", pushCommandName),
		Short:   "Push a local project to the remote cluster you're connected to",
		Long:    `Push a local project to the remote cluster you're connected to.`,
		Example: fmt.Sprintf(pushExample, cmdutil.CommandName(pushCommandName, fullParentName)),
		Args:    cobra.NoArgs,
	}
	options := pushOptions{commonOptions: &commonOptions{}}
	cmdutil.ConfigureRunnableAndCommandWithTargeting(&options, push)
	push.Flags().BoolVarP(&options.binary, "binary", "b", false, "Push packaged binary instead of source code")
	return push
}

func fetchPod(instance *component.Component) (*k8score.Pod, error) {
	client := k8s.GetClient()
	pods, err := client.KubeClient.CoreV1().Pods(instance.Namespace).List(v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"app": instance.Name}).String(),
	})
	if err != nil {
		return nil, err
	} else {
		// We assume that there is only one Pod containing the label app=component name AND we return it
		if len(pods.Items) > 0 {
			return &pods.Items[0], nil
		} else {
			err := fmt.Errorf("failed to get pod created for the component")
			return nil, err
		}
	}
}
