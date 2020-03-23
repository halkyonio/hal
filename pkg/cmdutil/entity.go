package cmdutil

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/ui"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"strings"
)

type ResourceType string

const (
	Component  ResourceType = "component"
	Capability ResourceType = "capability"
)

func (r ResourceType) String() string {
	return string(r)
}

func KnownResourceTypes() []ResourceType {
	return []ResourceType{Capability, Component}
}

func ResourceTypeFor(object runtime.Object) (ResourceType, error) {
	if object == nil {
		return "", fmt.Errorf("must provide a non-nil runtime.Object")
	}
	kind := strings.ToLower(object.GetObjectKind().GroupVersionKind().Kind)
	for _, t := range KnownResourceTypes() {
		if kind == t.String() {
			return t, nil
		}
	}
	return "", fmt.Errorf("unknown resource type: %s", kind)
}

type GenericOperationOptions struct {
	ResourceType  ResourceType
	Name          string
	Client        HalkyonEntity
	operationName string
	delegate      Runnable
}

// Exists checks if the object associated with this GenericOperationOptions exists. Returns (true, nil) if it exists,
// (false, nil) if it is determined to not exist or (false, error) if an error that doesn't determine existence occurred
func (o *GenericOperationOptions) Exists() (bool, error) {
	_, err := o.Client.Get(o.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func (o *GenericOperationOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	return o.delegate.Complete(name, cmd, args)
}

func (o *GenericOperationOptions) Validate() error {
	return o.delegate.Validate()
}

func (o *GenericOperationOptions) Run() error {
	return o.delegate.Run()
}

func (o *GenericOperationOptions) example(fullParentName string) string {
	tmpl := ktemplates.Examples(`  # %[1]s the %[2]s named 'foo'
  %[3]s foo`)
	return fmt.Sprintf(tmpl, strings.Title(o.operationName), o.ResourceType, CommandName(o.operationName, fullParentName))
}

func (o *GenericOperationOptions) use() string {
	return fmt.Sprintf("%s <name of the %s to %s>", o.operationName, o.ResourceType, o.operationName)
}

func (o *GenericOperationOptions) short() string {
	return fmt.Sprintf("%s the named %s", strings.Title(o.operationName), o.ResourceType)
}

func NewGenericOperation(fullParentName string, o *GenericOperationOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     o.use(),
		Short:   o.short(),
		Example: o.example(fullParentName),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			GenericRun(o, cmd, args)
		},
	}
	return cmd
}

type HalkyonEntity interface {
	Get(name string) (runtime.Object, error)
	Create(runtime.Object) error
	Delete(string, *v1.DeleteOptions) error
	GetKnown() ui.DisplayableMap
	GetNamespace() string
}
