package cmdutil

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/validation"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"reflect"
)

type TargetingOptions struct {
	paths      []string
	TargetPath string
	TargetName string
	runnable   Runnable
}

type withTargeting interface {
	SetTargetingOptions(options *TargetingOptions)
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

func NewTargetingOptions() *TargetingOptions {
	return &TargetingOptions{}
}

func (o *TargetingOptions) Complete(name string, cmd *cobra.Command, args []string) error {
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
			o.TargetPath = path
			o.TargetName = filepath.Base(o.TargetPath)
			err := o.runnable.Complete(name, cmd, args)
			if err != nil {
				return err
			}
		}
	} else {
		o.TargetPath = currentDir
		o.TargetName = filepath.Base(o.TargetPath)
	}

	return nil
}

func (o *TargetingOptions) Validate() error {
	return o.runForEachPath(o.runnable.Validate)
}

func (o *TargetingOptions) Run() error {
	return o.runForEachPath(o.runnable.Run)
}

func (o *TargetingOptions) runForEachPath(fn func() error) error {
	if len(o.paths) > 0 {
		for _, path := range o.paths {
			o.TargetPath = path
			o.TargetName = filepath.Base(o.TargetPath)
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

func (o *TargetingOptions) AttachFlagTo(cmd *cobra.Command) {
	cmd.Flags().StringSliceVarP(&o.paths, "target", "t", nil, "Execute the command on the target directories instead of the current one")
}
