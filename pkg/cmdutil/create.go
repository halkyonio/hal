package cmdutil

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
	"halkyon.io/hal/pkg/validation"
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
	Set(object runtime.Object)
}

type CreateOptions struct {
	*GenericOperationOptions
	Delegate       Creator
	fromDescriptor bool
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

func getComponentNames(components map[string]ComponentInfo) []string {
	names := make([]string, 0, len(components))
	for s := range components {
		names = append(names, s)
	}
	return names
}

func (o *CreateOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		o.Name = args[0]
	}

	// check if the user wants to use a component from a descriptor
	components, err := GetComponents()
	if err != nil {
		return err
	}
	componentsNb := len(components)
	if componentsNb > 0 {
		if componentsNb == 1 {
			o.Name = getComponentNames(components)[0]
		} else {
			if IsInteractive(cmd) && ui.Proceed(fmt.Sprintf("%d detected component(s), do you want to use one of them", componentsNb)) {
				o.Name = ui.Select("Component", getComponentNames(components), o.Name)
			}
		}
		info, ok := components[o.Name]
		if ok {
			ui.OutputSelection("Selected component from "+info.descriptor, o.Name)
			o.fromDescriptor = true
			// set the component on the delegate so it uses it when we want ask to create it
			o.Delegate.Set(info.component)
			exists, err := o.Exists()
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("A %s named '%s' already exists, please use update instead", o.ResourceType, o.Name)
			}
		}
	}

	if err = o.Delegate.Complete(name, cmd, args); err != nil {
		return err
	}

	for {
		o.Name = ui.Ask("Name", o.Name, o.generateName())
		err := validation.NameValidator(o.Name)
		if err != nil {
			ui.OutputError(fmt.Sprintf("Invalid name: '%s', please select another one", o.Name))
			o.Name = ""
		}
		exists, err := o.Exists()
		if err != nil {
			return err
		}
		if exists {
			ui.OutputError(fmt.Sprintf("A %s named '%s' already exists, please select another name", o.ResourceType, o.Name))
			o.Name = "" // reset name and try again!
		} else {
			break // resource is not found which is what we want
		}
	}

	return nil
}

func (o *CreateOptions) Exists() (bool, error) {
	err := o.Client.Get(o.Name, v1.GetOptions{})
	if err != nil {
		if util.IsKeyNotFoundError(errors.Cause(err)) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func (o *CreateOptions) Validate() error {
	if !o.fromDescriptor {
		if err := o.Delegate.Validate(); err != nil {
			return err
		}
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
