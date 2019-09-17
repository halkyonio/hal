package component

import (
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
)

func NewCmdDelete(fullParentName string) *cobra.Command {
	c := k8s.GetClient()
	generic := &cmdutil.DeleteOptions{
		ResourceType: "component",
		Client: client{
			client: c.HalkyonComponentClient.Components(c.Namespace),
			ns:     c.Namespace,
		},
	}
	return cmdutil.NewGenericDelete(fullParentName, generic)
}
