package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/kreate/pkg/kreate/cli/link"
	"halkyon.io/kreate/pkg/kreate/cli/mode"
	"halkyon.io/kreate/pkg/kreate/cli/project"
	"halkyon.io/kreate/pkg/kreate/cli/push"
)

const commandName = "hal"

func NewCmdKreate(version, commit, date string) *cobra.Command {
	kreate := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Easily create Kubernetes applications",
		Long: fmt.Sprintf(`%s %s built on '%s' (commit: %s)
Easily create and manage Kubernetes applications using Dekorate and the Halkyon operator, made with ❤️ by the Snowdrop team.`, commandName, version, date, commit),
	}

	kreate.AddCommand(
		project.NewCmdProject(commandName),
		push.NewCmdPush(commandName),
		mode.NewCmdMode(commandName),
		link.NewCmdLink(commandName),
	)

	return kreate
}
