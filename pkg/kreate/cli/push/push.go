package push

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const commandName = "push"

type options struct {
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	panic("implement me")
}

func (o *options) Validate() error {
	panic("implement me")
}

func (o *options) Run() error {
	panic("implement me")
}

func NewCmdPush(parent string) *cobra.Command {
	p := &options{}

	push := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Push a local project to the remote cluster you're connected to",
		Long:  `Push a local project to the remote cluster you're connected to.`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.GenericRun(p, cmd, args)
		},
	}

	return push
}
