package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/kreate/pkg/kreate/cli/link"
	"halkyon.io/kreate/pkg/kreate/cli/mode"
	"halkyon.io/kreate/pkg/kreate/cli/project"
	"halkyon.io/kreate/pkg/kreate/cli/push"
)

const commandName = "kreate"

func NewCmdKreate() *cobra.Command {
	kreate := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Easily create Kubernetes applications",
		Long:  `Easily create and manage Kubernetes applications using the Component operator created by the Snowdrop team.`,
	}

	kreate.AddCommand(
		project.NewCmdProject(commandName),
		push.NewCmdPush(commandName),
		mode.NewCmdMode(commandName),
		link.NewCmdLink(commandName),
	)

	return kreate
}
