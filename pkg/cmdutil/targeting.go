package cmdutil

import (
	"fmt"
	"github.com/spf13/cobra"
	halkyon "halkyon.io/api"
	"halkyon.io/api/component/v1beta1"
	"halkyon.io/kreate/pkg/validation"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8yml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"path/filepath"
	"reflect"
)

type ComponentTargetingOptions struct {
	paths          []string
	ComponentPath  string
	ComponentName  string
	DescriptorPath string
	runnable       Runnable
	deserializer   runtime.Decoder
}

type withTargeting interface {
	SetTargetingOptions(options *ComponentTargetingOptions)
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
	s := scheme.Scheme
	if err := halkyon.AddToScheme(s); err != nil {
		panic(err)
	}

	deserializer := scheme.Codecs.UniversalDeserializer()

	return &ComponentTargetingOptions{deserializer: deserializer}
}

func (o *ComponentTargetingOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	// todo: separate component identification logic from path / directory
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(o.paths) > 0 {
		for i, path := range o.paths {
			path = filepath.Join(currentDir, path)
			if !validation.CheckFileExist(path) {
				return fmt.Errorf("%s doesn't exist", path)
			}
			o.paths[i] = path
			o.ComponentPath = path
			if err := o.initDescriptorPath(); err != nil {
				return err
			}
			if err := o.initComponentName(); err != nil {
				return err
			}
			err := o.runnable.Complete(name, cmd, args)
			if err != nil {
				return err
			}
		}
	} else {
		o.ComponentPath = currentDir
		if err := o.initDescriptorPath(); err != nil {
			return err
		}
		if err := o.initComponentName(); err != nil {
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
	if len(o.paths) > 0 {
		for _, path := range o.paths {
			o.ComponentPath = path
			o.ComponentName = filepath.Base(o.ComponentPath)
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

func (o *ComponentTargetingOptions) initDescriptorPath() error {
	descriptor := filepath.Join(o.ComponentPath, "target", "classes", "META-INF", "dekorate", "halkyon.yml")
	if !validation.CheckFileExist(descriptor) {
		return fmt.Errorf("halkyon descriptor was not found at %s", descriptor)
	}
	o.DescriptorPath = descriptor
	return nil
}

func (o *ComponentTargetingOptions) initComponentName() error {
	file, err := os.Open(o.DescriptorPath)
	if err != nil {
		return err
	}

	decoder := k8yml.NewYAMLToJSONDecoder(file)
	list := &v1.List{}
	err = decoder.Decode(list)
	for _, value := range list.Items {
		object := value.Object
		if object == nil {
			object, _, err = o.deserializer.Decode(value.Raw, nil, nil)
			if err != nil {
				return err
			}
		}
		if c, ok := object.(*v1beta1.Component); ok {
			o.ComponentName = c.Name
			return nil
		}
	}
	return fmt.Errorf("no component configuration found in %s", o.DescriptorPath)
}

func (o *ComponentTargetingOptions) AttachFlagTo(cmd *cobra.Command) {
	cmd.Flags().StringSliceVarP(&o.paths, "components", "c", nil, "Execute the command on the target component(s) instead of the current one")
}
