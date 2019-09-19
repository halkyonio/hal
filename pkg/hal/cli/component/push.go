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
	"k8s.io/apimachinery/pkg/types"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"os"
	"path/filepath"
	"strings"
)

const pushCommandName = "push"

type pushOptions struct {
	*commonOptions
	source bool
}

var (
	pushExample = ktemplates.Examples(`  # Deploy the components client-sb, backend-sb
  %[1]s -c client-sb,backend-sb`)
	excludedFileNames = map[string]bool{
		"target":    true,
		".git":      true,
		".DS_Store": true,
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
	file, err := os.Open(binaryPath)
	if err != nil {
		if o.source && os.IsNotExist(err) {
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

			tar := archiver.NewTar()
			tar.OverwriteExisting = true
			if err := tar.Archive(toTar, binaryPath); err != nil {
				return err
			}
		} else {
			return err
		}
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
	return len(podName) > 0 && !k8s.IsJarPresent(podName)
}

func (o *pushOptions) push(component *component.Component) error {
	c := k8s.GetClient()
	podName := component.Status.PodName
	toPush, _ := o.getComponentBinaryPath()
	s := log.Spinner("Uploading " + toPush)
	defer s.End(false)
	err := k8s.Copy(toPush, c.Namespace, podName, o.source)
	if err != nil {
		return fmt.Errorf("error uploading file: %v", err)
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

	if o.source {
		err = c.ExecCMDInContainer(podName, []string{"tar", "xmf", k8s.SourcePathInContainer, "-C", k8s.ExtractedSourcePathInContainer}, pipeWriter, pipeWriter, nil, false)
		if err != nil {
			return err
		}

		err = c.ExecCMDInContainer(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "start", "build"}, pipeWriter, pipeWriter, nil, false)
		if err != nil {
			return err
		}
	}

	err = c.ExecCMDInContainer(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "stop", "run"}, pipeWriter, pipeWriter, nil, false)
	if err != nil {
		return err
	}
	err = c.ExecCMDInContainer(podName, []string{"/var/lib/supervisord/bin/supervisord", "ctl", "start", "run"}, pipeWriter, pipeWriter, nil, false)
	if err != nil {
		return err
	}
	return nil
}

func (o *pushOptions) getComponentBinaryPath() (string, error) {
	if o.source {
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
	push.Flags().BoolVarP(&options.source, "source", "s", false, "Push source code instead of packaged binary")
	return push
}
