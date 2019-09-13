package cmdutil

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record/util"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const deleteCommandName = "delete"

type DeleteOptions struct {
	ResourceType string
	Name         string
	Client       HalkyonEntity
}

type HalkyonEntity interface {
	Get(string, v1.GetOptions) error
	Delete(string, *v1.DeleteOptions) error
	GetKnown() []string
	GetNamespace() string
}

var (
	Example = ktemplates.Examples(`  # Delete the %[2]s named 'foo'
  %[1]s foo`)
)

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
	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s <name of %s to delete>", deleteCommandName, o.ResourceType),
		Short:   fmt.Sprintf("Delete the named %s", o.ResourceType),
		Long:    fmt.Sprintf("Delete the named %s if it exists", o.ResourceType),
		Example: fmt.Sprintf(Example, CommandName(deleteCommandName, fullParentName), o.ResourceType),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			GenericRun(o, cmd, args)
		},
	}
	return cmd
}
