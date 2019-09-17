package capability

import (
	"github.com/spf13/cobra"
	"halkyon.io/api/capability/clientset/versioned/typed/capability/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdDelete(fullParentName string) *cobra.Command {
	c := k8s.GetClient()
	generic := &cmdutil.DeleteOptions{
		ResourceType: "capability",
		Client: client{
			client: c.HalkyonCapabilityClient.Capabilities(c.Namespace),
			ns:     c.Namespace,
		},
	}
	return cmdutil.NewGenericDelete(fullParentName, generic)
}
