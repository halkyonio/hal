package cmdutil

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
	"halkyon.io/hal/pkg/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record/util"
	"os"
	"path/filepath"
	"time"
)

const createCommandName = "create"

type Creator interface {
	Runnable
	GeneratePrefix() string
	Build() runtime.Object
	Set(entity runtime.Object)
}

type CreateOptions struct {
	*GenericOperationOptions
	Delegate       Creator
	fromDescriptor bool
}

func NewCreateOptions(resourceType ResourceType, client HalkyonEntity) *CreateOptions {
	c := &CreateOptions{}
	c.GenericOperationOptions = &GenericOperationOptions{
		ResourceType:  resourceType,
		Client:        client,
		operationName: createCommandName,
		delegate:      c,
	}
	return c
}

func entityNames(registry entitiesRegistry) []string {
	names := make([]string, 0, len(registry))
	for _, entity := range registry {
		names = append(names, entity.Name)
	}
	return names
}

func (o *CreateOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		o.Name = args[0]
	}

	// look for locally existing components that also don't already exist on the remote cluster
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	hd := LoadAvailableHalkyonEntities(currentDir)
	entities := hd.GetDefinedEntitiesWith(o.ResourceType)
	size := len(entities)
	t := o.ResourceType.String()
	if size > 0 {
		// if we have an entity with the same name as the current directory, use it by default
		currentDirName := filepath.Base(currentDir)
		if entity, ok := entities[currentDirName]; ok {
			o.Name = entity.Name
			o.fromDescriptor = true
			o.Delegate.Set(entity.Entity)
		} else {
			names := entityNames(entities)
			if size == 1 {
				if o.Name == names[0] {
					entity := entities[o.Name]
					if IsInteractive(cmd) && ui.Proceed(fmt.Sprintf("Found %s named %s in %s, use it", t, o.Name, entity.Path)) {
						o.Name = entity.Name
						o.fromDescriptor = true
						o.Delegate.Set(entity.Entity)
					}
				}
			} else if IsInteractive(cmd) && ui.Proceed(fmt.Sprintf("Found %d %s(s) in %s, do you want to %s from them", size, t, currentDirName, o.operationName)) {
				o.Name = ui.Select(t, names, o.Name)
			}
		}

	}

	entity, ok := entities[o.Name]
	if ok {
		ui.OutputSelection(fmt.Sprintf("Selected %s from %s", t, entity.Path), o.Name)
		o.fromDescriptor = true
		// set the component on the delegate so it uses it when we want ask to create it
		o.Delegate.Set(entity.Entity)
		exists, err := o.Exists()
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("a %s named '%s' already exists, please use update instead (NOT YET IMPLEMENTED)", o.ResourceType, o.Name)
		}
	}

	if !o.fromDescriptor {
		if err := o.Delegate.Complete(name, cmd, args); err != nil {
			return err
		}
	}

	for {
		o.Name = ui.Ask("Name", o.Name, o.generateName())
		err := validation.NameValidator(o.Name)
		if err != nil {
			ui.OutputError(fmt.Sprintf("Invalid name: '%s', please select another one", o.Name))
			o.Name = ""
		}
		_, err = o.Client.Get(o.Name)
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

func (o *CreateOptions) Exists() (bool, error) {
	_, err := o.Client.Get(o.Name)
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
