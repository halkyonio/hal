package cmdutil

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/io"
)

type Runnable interface {
	Complete(name string, cmd *cobra.Command, args []string) error
	Validate() error
	Run() error
}

func GenericRun(o Runnable, cmd *cobra.Command, args []string) {
	io.LogErrorAndExit(o.Complete(cmd.Name(), cmd, args), fmt.Sprintf("error completing %s", cmd.Name()))
	io.LogErrorAndExit(o.Validate(), fmt.Sprintf("error validating %s", cmd.Name()))
	io.LogErrorAndExit(o.Run(), fmt.Sprintf("error running %s", cmd.Name()))
}
