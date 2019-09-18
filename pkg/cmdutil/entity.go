package cmdutil

import (
	"fmt"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"strings"
)

type GenericOperationOptions struct {
	ResourceType  string
	Name          string
	Client        HalkyonEntity
	operationName string
	delegate      Runnable
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
	Get(string, v1.GetOptions) error
	Create(runtime.Object) error
	Delete(string, *v1.DeleteOptions) error
	GetKnown() []string
	GetNamespace() string
}
