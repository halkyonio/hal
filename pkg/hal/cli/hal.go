package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/hal/cli/component"
	"halkyon.io/hal/pkg/hal/cli/link"
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
		component.NewCmdComponent(commandName),
		link.NewCmdLink(commandName),
		version.NewCmdVersion(commandName),
	)

	return hal
}
