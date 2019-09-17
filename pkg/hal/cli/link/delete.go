package link

import (
	"github.com/spf13/cobra"
	"halkyon.io/api/link/clientset/versioned/typed/link/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdDelete(fullParentName string) *cobra.Command {
	c := k8s.GetClient()
	generic := &cmdutil.DeleteOptions{
		ResourceType: "link",
		Client: client{
			client: c.HalkyonLinkClient.Links(c.Namespace),
			ns:     c.Namespace,
		},
	}
	return cmdutil.NewGenericDelete(fullParentName, generic)
}
