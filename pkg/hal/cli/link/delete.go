package link

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"halkyon.io/api/link/clientset/versioned/typed/link/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record/util"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const deleteCommandName = "delete"

type deleteOptions struct {
	name   string
	ns     string
	client v1beta1.LinkInterface
}

var (
	createExample = ktemplates.Examples(`  # Delete the link named 'foo'
  %[1]s foo`)
)

func (o *deleteOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		o.name = ""
	} else {
		o.name = args[0]
	}
	c := k8s.GetClient()
	o.ns = c.Namespace
	o.client = c.HalkyonLinkClient.Links(o.ns)
	return nil
}

func (o *deleteOptions) Validate() error {
	needLinkName := len(o.name) == 0
	if !needLinkName {
		_, err := o.client.Get(o.name, v1.GetOptions{})
		if err != nil {
			if util.IsKeyNotFoundError(errors.Cause(err)) {
				needLinkName = true
			} else {
				return err
			}
		}
	}
	if needLinkName {
		links := o.getKnownLinks()
		if len(links) == 0 {
			return fmt.Errorf("no link currently exist in '%s'", o.ns)
		}
		s := "Unknown link"
		if len(o.name) == 0 {
			s = "No provided link name"
		}
		message := ui.SelectFromOtherErrorMessage(s, o.name)
		o.name = ui.Select(message, links)
	}
	return nil
}

func (o *deleteOptions) getKnownLinks() []string {
	list, err := o.client.List(v1.ListOptions{})
	if err != nil {
		return []string{}
	}
	links := list.Items
	names := make([]string, 0, len(links))
	for _, link := range links {
		names = append(names, link.Name)
	}
	return names
}

func (o *deleteOptions) Run() error {
	if ui.Proceed(fmt.Sprintf("Really delete '%s' link", o.name)) {
		err := o.client.Delete(o.name, &v1.DeleteOptions{})
		if err == nil {
			log.Successf("Successfully deleted '%s' link", o.name)
		}
		return err
	}
	log.Errorf("Canceled deletion of '%s' link", o.name)
	return nil
}

func NewCmdDelete(fullParentName string) *cobra.Command {
	o := &deleteOptions{}
	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s <name of link to delete>", deleteCommandName),
		Short:   "Delete the named link",
		Long:    `Delete the named link if it exists`,
		Example: fmt.Sprintf(createExample, cmdutil.CommandName(deleteCommandName, fullParentName)),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.GenericRun(o, cmd, args)
		},
	}
	return cmd
}
