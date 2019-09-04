package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/hal/cli/link"
	"halkyon.io/hal/pkg/hal/cli/mode"
	"halkyon.io/hal/pkg/hal/cli/project"
	"halkyon.io/hal/pkg/hal/cli/push"
)

const commandName = "hal"

func NewCmdKreate(version, commit, date string) *cobra.Command {
	hal := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Easily create Kubernetes applications",
		Long: fmt.Sprintf(`%s %s built on '%s' (commit: %s)
Easily create and manage Kubernetes applications using Dekorate and the Halkyon operator, made with ❤️ by the Snowdrop team.`, commandName, version, date, commit),
	}

	hal.AddCommand(
		project.NewCmdProject(commandName),
		push.NewCmdPush(commandName),
		mode.NewCmdMode(commandName),
		link.NewCmdLink(commandName),
	)

	return hal
}
