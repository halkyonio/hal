package cmdutil

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record/util"
	"time"
)

const createCommandName = "create"

type Creator interface {
	Runnable
	GeneratePrefix() string
	Build() runtime.Object
}

type CreateOptions struct {
	*GenericOperationOptions
	Delegate Creator
}

func NewCreateOptions(resourceType string, client HalkyonEntity) *CreateOptions {
	c := &CreateOptions{}
	c.GenericOperationOptions = &GenericOperationOptions{
		ResourceType:  resourceType,
		Client:        client,
		operationName: createCommandName,
		delegate:      c,
	}
	return c
}

func (o *CreateOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	if err := o.Delegate.Complete(name, cmd, args); err != nil {
		return err
	}

	if len(args) == 1 {
		o.Name = args[0]
	}
	for {
		o.Name = ui.Ask("Name", o.Name, o.generateName())
		err := o.Client.Get(o.Name, v1.GetOptions{})
		if err != nil {
			if util.IsKeyNotFoundError(errors.Cause(err)) {
				break // resource is not found which is what we want
			} else {
				return err
			}
		} else {
			ui.OutputError(fmt.Sprintf("A %s named '%s' already exists, please select another name", o.ResourceType, o.Name))
			o.Name = "" // reset name and try again!
		}
	}

	return nil
}

func (o *CreateOptions) Validate() error {
	if err := o.Delegate.Validate(); err != nil {
		return err
	}
	return nil
}

func (o *CreateOptions) Run() error {
	err := o.Client.Create(o.Delegate.Build())
	if err == nil {
		log.Successf("Successfully created '%s' %s", o.Name, o.ResourceType)
	}
	return err
}

func (o *CreateOptions) generateName() string {
	return fmt.Sprintf("%s-%s-%d", o.Delegate.GeneratePrefix(), o.ResourceType, time.Now().UnixNano())
}

func NewGenericCreate(fullParentName string, o *CreateOptions) *cobra.Command {
	create := NewGenericOperation(fullParentName, o.GenericOperationOptions)
	return create
}
