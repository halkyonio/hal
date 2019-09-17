package cmdutil

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record/util"
)

const deleteCommandName = "delete"

type DeleteOptions struct {
	*HalkyonEntityOptions
}

func NewDeleteOptions(resourceType string, client HalkyonEntity) *DeleteOptions {
	d := &DeleteOptions{}
	d.HalkyonEntityOptions = &HalkyonEntityOptions{
		ResourceType:  resourceType,
		Client:        client,
		operationName: deleteCommandName,
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
	if !needName {
		err := o.Client.Get(o.Name, v1.GetOptions{})
		if err != nil {
			if util.IsKeyNotFoundError(errors.Cause(err)) {
				needName = true
			} else {
				return err
			}
		}
	}
	if needName {
		known := o.Client.GetKnown()
		if len(known) == 0 {
			return fmt.Errorf("no %s currently exist in '%s'", o.ResourceType, o.Client.GetNamespace())
		}
		s := "Unknown " + o.ResourceType
		if len(o.Name) == 0 {
			s = "No provided " + o.ResourceType + " name"
		}
		message := ui.SelectFromOtherErrorMessage(s, o.Name)
		o.Name = ui.Select(message, known)
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
	return NewGenericOperation(fullParentName, o.HalkyonEntityOptions)
}
