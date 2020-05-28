package component

import (
	"fmt"
	"halkyon.io/api/component/clientset/versioned/typed/component/v1beta1"
	v1beta12 "halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/ui"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type client struct {
	client v1beta1.ComponentInterface
	ns     string
}

var _ cmdutil.HalkyonEntity = client{}

func (lc client) Create(toCreate runtime.Object) error {
	component := toCreate.(*v1beta12.Component)
	c, err := lc.Get(component.Name)
	if errors.IsNotFound(err) {
		// create
		_, err = lc.client.Create(component)
	} else if err == nil {
		component.ResourceVersion = c.(*v1beta12.Component).ResourceVersion
		c, err = lc.client.Update(component)
	}

	return err
}

func (lc client) Get(name string) (runtime.Object, error) {
	return lc.GetTyped(name)
}

func (lc client) GetTyped(name string) (*v1beta12.Component, error) {
	return lc.client.Get(name, v1.GetOptions{})
}

type displayableCapability struct {
	c v1beta12.Component
}

var _ ui.Displayable = displayableCapability{}

func (d displayableCapability) Help() string {
	return fmt.Sprintf("%s (%v/%v)", d.Name(), d.c.Spec.Runtime, d.c.Spec.Version)
}

func (d displayableCapability) Display() string {
	return d.Help()
}

func (d displayableCapability) Name() string {
	return d.c.Name
}

func (d displayableCapability) GetUnderlying() interface{} {
	return d.c
}

func NewDisplayableCapability(capability v1beta12.Component) ui.Displayable {
	return displayableCapability{capability}
}

func (lc client) GetKnown() ui.DisplayableMap {
	list, err := lc.client.List(v1.ListOptions{})
	if err != nil {
		return ui.Empty
	}
	items := list.Items
	result := ui.NewDisplayableMap(len(items))
	for _, item := range items {
		result.Add(NewDisplayableCapability(item))
	}
	return result
}

func (lc client) Delete(name string, options *v1.DeleteOptions) error {
	return lc.client.Delete(name, options)
}

func (lc client) GetNamespace() string {
	return lc.ns
}

func typeMeta() v1.TypeMeta {
	return v1.TypeMeta{
		Kind:       v1beta12.Kind,
		APIVersion: "halkyon.io/v1beta1",
	}
}

var Entity client

func init() {
	c := k8s.GetClient()
	Entity = client{
		client: c.HalkyonComponentClient.Components(c.Namespace),
		ns:     c.Namespace,
	}
}
