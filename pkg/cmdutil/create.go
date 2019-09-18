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
	"strings"
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
	if len(args) == 0 {
		o.Name = ui.Ask("Name", o.Name, o.generateName())
	} else {
		o.Name = args[0]
	}

	return nil
}

func (o *CreateOptions) Validate() error {
	if err := o.Delegate.Validate(); err != nil {
		return err
	}

	err := o.Client.Get(o.Name, v1.GetOptions{})
	if err != nil {
		if util.IsKeyNotFoundError(errors.Cause(err)) {
			return nil
		} else {
			return err
		}
	}
	return fmt.Errorf("a %s named '%s' already exists, please select another name", o.ResourceType, o.Name)
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
	create.Flags().StringVarP(&o.Name, "name", "n", "", fmt.Sprintf("%s name", strings.Title(o.ResourceType)))
	return create
}
