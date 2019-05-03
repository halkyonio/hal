package cmdutil

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/validation"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

type TargetingOptions struct {
	Target string
}

func NewTargetingOptions() *TargetingOptions {
	return &TargetingOptions{}
}

func (o *TargetingOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(o.Target) > 0 {
		o.Target = filepath.Join(currentDir, o.Target)
		if !validation.CheckFileExist(o.Target) {
			return fmt.Errorf("%s doesn't exist", o.Target)
		}
	} else {
		o.Target = currentDir
	}
	return nil
}

func (o *TargetingOptions) Validate() error {
	return nil
}

func (o *TargetingOptions) Run() error {
	return nil
}

func (o *TargetingOptions) AttachFlagTo(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Target, "target", "t", "", "Execute the command on the target directory instead of the current one")
}
