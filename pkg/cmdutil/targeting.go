package cmdutil

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/validation"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

type TargetingOptions struct {
	TargetPath string
	TargetName string
}

func NewTargetingOptions() *TargetingOptions {
	return &TargetingOptions{}
}

func (o *TargetingOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(o.TargetPath) > 0 {
		o.TargetPath = filepath.Join(currentDir, o.TargetPath)
		if !validation.CheckFileExist(o.TargetPath) {
			return fmt.Errorf("%s doesn't exist", o.TargetPath)
		}
	} else {
		o.TargetPath = currentDir
	}

	o.TargetName = filepath.Base(o.TargetPath)

	return nil
}

func (o *TargetingOptions) Validate() error {
	return nil
}

func (o *TargetingOptions) Run() error {
	return nil
}

func (o *TargetingOptions) AttachFlagTo(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.TargetPath, "target", "t", "", "Execute the command on the target directory instead of the current one")
}
