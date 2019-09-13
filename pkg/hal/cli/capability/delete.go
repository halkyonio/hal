package capability

import (
	"github.com/spf13/cobra"
	"halkyon.io/api/capability/clientset/versioned/typed/capability/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type client struct {
	client v1beta1.CapabilityInterface
	ns     string
}

func (lc client) Get(name string, options v1.GetOptions) error {
	_, err := lc.client.Get(name, options)
	return err
}

func (lc client) GetKnown() []string {
	list, err := lc.client.List(v1.ListOptions{})
	if err != nil {
		return []string{}
	}
	items := list.Items
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func (lc client) Delete(name string, options *v1.DeleteOptions) error {
	return lc.client.Delete(name, options)
}

func (lc client) GetNamespace() string {
	return lc.ns
}

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
