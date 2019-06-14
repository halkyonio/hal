package cmdutil

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/validation"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"reflect"
)

type ComponentTargetingOptions struct {
	paths         []string
	ComponentPath string
	ComponentName string
	runnable      Runnable
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
	return &ComponentTargetingOptions{}
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
			o.ComponentName = filepath.Base(o.ComponentPath)
			err := o.runnable.Complete(name, cmd, args)
			if err != nil {
				return err
			}
		}
	} else {
		o.ComponentPath = currentDir
		o.ComponentName = filepath.Base(o.ComponentPath)
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

func (o *ComponentTargetingOptions) AttachFlagTo(cmd *cobra.Command) {
	cmd.Flags().StringSliceVarP(&o.paths, "components", "c", nil, "Execute the command on the target component(s) instead of the current one")
}
