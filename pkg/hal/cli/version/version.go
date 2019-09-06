package version

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
)

const commandName = "version"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	parent  = "hal"
)

type options struct {
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	return nil
}

func (o *options) Validate() error {
	return nil
}

func (o *options) Run() error {
	fmt.Println(Version())
	return nil
}

func NewCmdVersion(parentCmdName string) *cobra.Command {
	o := &options{}
	parent = parentCmdName
	version := &cobra.Command{
		Use:   fmt.Sprintf("%s", commandName),
		Short: "Displays this tool's version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.GenericRun(o, cmd, args)
		},
	}
	return version
}

func Version() string {
	return fmt.Sprintf("%s %s built with ❤️  by the Snowdrop team on '%s' (commit: %s)", parent, version, date, commit)
}
