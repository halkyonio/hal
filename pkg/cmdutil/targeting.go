package cmdutil

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/validation"
	"k8s.io/api/core/v1"
	k8yml "k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path/filepath"
	"reflect"
)

type ComponentTargetingOptions struct {
	paths    []string
	current  targetComponent
	targets  []targetComponent
	runnable Runnable
}

type targetComponent struct {
	path       string
	descriptor string
	name       string
}

func initTargetComponent(path string) (tc targetComponent, err error) {
	// check that we have an halkyon descriptor
	descriptor := halkyonDescriptorFrom(path)
	tc.name = filepath.Base(path)
	tc.path = path
	tc.descriptor = descriptor
	return tc, nil
}

func initTargetComponentFromDekorate(path string) (tc targetComponent, err error) {
	// check that we have an halkyon descriptor
	descriptor := halkyonDescriptorFrom(path)
	if !validation.CheckFileExist(descriptor) {
		return tc, fmt.Errorf("halkyon descriptor was not found at %s", descriptor)
	}

	// look for the component name in the halkyon descriptor
	file, err := os.Open(descriptor)
	if err != nil {
		return tc, err
	}
	decoder := k8yml.NewYAMLToJSONDecoder(file)
	list := &v1.List{}
	err = decoder.Decode(list)
	for _, value := range list.Items {
		object := value.Object
		if object == nil {
			object, _, err = deserializer.Decode(value.Raw, nil, nil)
			if err != nil {
				return tc, err
			}
		}
		// look for a component descriptor in the halkyon list
		if c, ok := object.(*v1beta1.Component); ok {
			tc.name = c.Name
			tc.path = path
			tc.descriptor = descriptor
			return tc, nil
		}
	}
	return tc, fmt.Errorf("no component configuration found in %s", descriptor)
}

type withTargeting interface {
	SetTargetingOptions(options *ComponentTargetingOptions)
}

func (o *ComponentTargetingOptions) GetTargetedComponentPath() string {
	return o.current.path
}

func (o *ComponentTargetingOptions) GetTargetedComponentDescriptor() string {
	return o.current.descriptor
}

func (o *ComponentTargetingOptions) GetTargetedComponentName() string {
	return o.current.name
}

func ConfigureRunnableAndCommandWithTargeting(runnable Runnable, cmd *cobra.Command) {
	if targeting, ok := runnable.(withTargeting); ok {
		targetingOptions := NewTargetingOptions()
		targetingOptions.AttachFlagTo(cmd)
		targetingOptions.runnable = runnable
		cmd.Run = func(cmd *cobra.Command, args []string) {
			GenericRun(targetingOptions, cmd, args)
		}
		targeting.SetTargetingOptions(targetingOptions)
	} else {
		panic(fmt.Errorf("provided Runnable %s doesn't implement withTargeting interface", reflect.TypeOf(runnable)))
	}
}

func NewTargetingOptions() *ComponentTargetingOptions {
	return &ComponentTargetingOptions{}
}

func (o *ComponentTargetingOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	// todo: separate component identification logic from path / directory
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	targetNb := len(o.paths)
	if targetNb > 0 {
		o.targets = make([]targetComponent, 0, targetNb)
		for _, path := range o.paths {
			path = filepath.Join(currentDir, path)
			if !validation.IsValidDir(path) {
				return fmt.Errorf("%s doesn't exist", path)
			}
			// set current target
			o.current, err = initTargetComponent(path)
			if err != nil {
				return err
			}
			// record target for later reuse
			o.targets = append(o.targets, o.current)
			err := o.runnable.Complete(name, cmd, args)
			if err != nil {
				return err
			}
		}
	} else {
		o.current, err = initTargetComponent(currentDir)
		if err != nil {
			return err
		}
		return o.runnable.Complete(name, cmd, args)
	}

	return nil
}

func (o *ComponentTargetingOptions) Validate() error {
	return o.runForEachPath(o.runnable.Validate)
}

func (o *ComponentTargetingOptions) Run() error {
	return o.runForEachPath(o.runnable.Run)
}

func (o *ComponentTargetingOptions) runForEachPath(fn func() error) error {
	if len(o.targets) > 0 {
		for _, target := range o.targets {
			o.current = target
			err := fn()
			if err != nil {
				return err
			}
		}
	} else {
		return fn()
	}
	return nil
}

func (o *ComponentTargetingOptions) AttachFlagTo(cmd *cobra.Command) {
	cmd.Flags().StringSliceVarP(&o.paths, "components", "c", nil, "Execute the command on the target component(s) instead of the current one")
}
