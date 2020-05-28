package cmdutil

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const deleteCommandName = "delete"

type DeleteOptions struct {
	*GenericOperationOptions
}

func NewDeleteOptions(resourceType ResourceType, client HalkyonEntity) *DeleteOptions {
	d := &DeleteOptions{}
	d.GenericOperationOptions = &GenericOperationOptions{
		ResourceType:  resourceType,
		Client:        client,
		OperationName: deleteCommandName,
		delegate:      d,
	}
	return d
}

func (o *DeleteOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		o.Name = ""
	} else {
		o.Name = args[0]
	}

	return nil
}

func (o *DeleteOptions) Validate() error {
	needName := len(o.Name) == 0
	// if a name is provided, check that it corresponds to an existing component
	if found, err := o.Exists(); !needName {
		if !found {
			if err == nil {
				needName = true
			} else {
				return err
			}
		}
	}
	if needName {
		known := o.Client.GetKnown()
		if known.Len() == 0 {
			return fmt.Errorf("no %s currently exist in '%s'", o.ResourceType, o.Client.GetNamespace())
		}
		s := "Unknown " + o.ResourceType
		if len(o.Name) == 0 {
			s = "No provided " + o.ResourceType + " name"
		}
		message := ui.SelectFromOtherErrorMessage(s.String(), o.Name)
		o.Name = ui.SelectDisplayable(message, known).Name()
	}
	return nil
}

func (o *DeleteOptions) Run() error {
	if ui.Proceed(fmt.Sprintf("Really delete '%s' %s", o.Name, o.ResourceType)) {
		err := o.Client.Delete(o.Name, &v1.DeleteOptions{})
		if err == nil {
			log.Successf("Successfully deleted '%s' %s", o.Name, o.ResourceType)
		}
		return err
	}
	log.Errorf("Canceled deletion of '%s' %s", o.Name, o.ResourceType)
	return nil
}

func NewGenericDelete(fullParentName string, o *DeleteOptions) *cobra.Command {
	return NewGenericOperation(fullParentName, o.GenericOperationOptions)
}
