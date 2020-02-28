package capability

import (
	"halkyon.io/api/capability/clientset/versioned/typed/capability/v1beta1"
	v1beta12 "halkyon.io/api/capability/v1beta1"
	"halkyon.io/hal/pkg/k8s"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type client struct {
	client v1beta1.CapabilityInterface
	ns     string
}

func (lc client) Create(toCreate runtime.Object) error {
	_, err := lc.client.Create(toCreate.(*v1beta12.Capability))
	return err
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

func (lc client) GetMatching(spec ...v1beta12.CapabilitySpec) map[string]v1beta12.CapabilitySpec {
	list, err := lc.client.List(v1.ListOptions{})
	if err != nil {
		return map[string]v1beta12.CapabilitySpec{}
	}
	items := list.Items
	matching := make(map[string]v1beta12.CapabilitySpec, len(items))
	skipMatch := len(spec) != 1
	for _, item := range items {
		if skipMatch || item.Spec.Matches(spec[0]) {
			matching[item.Name] = item.Spec
		}
	}
	return matching
}

func (lc client) Delete(name string, options *v1.DeleteOptions) error {
	return lc.client.Delete(name, options)
}

func (lc client) GetNamespace() string {
	return lc.ns
}

var Entity client

func init() {
	c := k8s.GetClient()
	Entity = client{
		client: c.HalkyonCapabilityClient.Capabilities(c.Namespace),
		ns:     c.Namespace,
	}
}
