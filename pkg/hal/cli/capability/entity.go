package capability

import (
	"fmt"
	"halkyon.io/api/capability/clientset/versioned/typed/capability/v1beta1"
	v1beta12 "halkyon.io/api/capability/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/ui"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type client struct {
	client v1beta1.CapabilityInterface
	ns     string
}

var _ cmdutil.HalkyonEntity = &client{}

func (lc client) Create(toCreate runtime.Object) error {
	_, err := lc.client.Create(toCreate.(*v1beta12.Capability))
	return err
}

func (lc client) Get(name string) (runtime.Object, error) {
	return lc.GetTyped(name)
}

func (lc client) GetTyped(name string) (*v1beta12.Capability, error) {
	return lc.client.Get(name, v1.GetOptions{})
}

func (lc client) GetKnown() ui.DisplayableMap {
	return lc.GetMatching()
}

type displayableCapability struct {
	capability v1beta12.Capability
}

var _ ui.Displayable = displayableCapability{}

func (d displayableCapability) Help() string {
	return GetDisplay(d.Name(), d.capability.Spec)
}

func GetDisplay(name string, spec v1beta12.CapabilitySpec) string {
	return fmt.Sprintf("%s (%v/%v/%s)", name, spec.Category, spec.Type, spec.Version)
}

func (d displayableCapability) Display() string {
	return d.Help()
}

func (d displayableCapability) Name() string {
	return d.capability.Name
}

func (d displayableCapability) GetUnderlying() interface{} {
	return d.capability
}

func NewDisplayableCapability(capability v1beta12.Capability) ui.Displayable {
	return displayableCapability{capability}
}

func (lc client) GetMatching(spec ...v1beta12.CapabilitySpec) ui.DisplayableMap {
	r := make(chan ui.DisplayableMap)

	go func() {
		list, err := lc.client.List(v1.ListOptions{})
		if err != nil {
			r <- ui.Empty
			return
		}
		items := list.Items
		result := ui.NewDisplayableMap(len(items))
		skipMatch := len(spec) != 1
		for _, item := range items {
			if skipMatch || item.Spec.Matches(spec[0]) {
				result.Add(NewDisplayableCapability(item))
			}
		}
		r <- result
	}()

	return <-r
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
