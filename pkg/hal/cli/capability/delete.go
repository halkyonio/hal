package capability

import (
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
)

func NewCmdDelete(fullParentName string) *cobra.Command {
	c := k8s.GetClient()
	generic := cmdutil.NewDeleteOptions("capability", client{
		client: c.HalkyonCapabilityClient.Capabilities(c.Namespace),
		ns:     c.Namespace,
	})
	return cmdutil.NewGenericDelete(fullParentName, generic)
}
