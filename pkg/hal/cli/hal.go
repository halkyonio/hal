package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/hal/cli/link"
	"halkyon.io/hal/pkg/hal/cli/mode"
	"halkyon.io/hal/pkg/hal/cli/project"
	"halkyon.io/hal/pkg/hal/cli/push"
	"halkyon.io/hal/pkg/hal/cli/version"
)

const commandName = "hal"

func NewCmdHal() *cobra.Command {
	hal := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Easily create Kubernetes applications",
		Long: fmt.Sprintf(`%s
Easily create and manage Kubernetes applications using Dekorate and the Halkyon operator.`, version.Version()),
	}

	hal.AddCommand(
		project.NewCmdProject(commandName),
		push.NewCmdPush(commandName),
		mode.NewCmdMode(commandName),
		link.NewCmdLink(commandName),
		version.NewCmdVersion(commandName),
	)

	return hal
}
