package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/hal/cli/capability"
	"halkyon.io/hal/pkg/hal/cli/component"
	"halkyon.io/hal/pkg/hal/cli/version"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const commandName = "hal"

var (
	halExample = ktemplates.Examples(`  # Displays hal help
 %[1]s  --help`)
)

func NewCmdHal() *cobra.Command {
	hal := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Easily create Kubernetes applications",
		Long: fmt.Sprintf(`%s
Easily create and manage Kubernetes applications using Dekorate and the Halkyon operator.`, version.Version()),
		Example: fmt.Sprintf(halExample, commandName),
	}

	hal.AddCommand(
		capability.NewCmdCapability(commandName),
		component.NewCmdComponent(commandName),
		version.NewCmdVersion(commandName),
	)

	return hal
}
