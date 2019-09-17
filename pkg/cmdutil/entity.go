package cmdutil

import (
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"strings"
)

type HalkyonEntityOptions struct {
	ResourceType string
	Name         string
	Client       HalkyonEntity
}

func (o *HalkyonEntityOptions) genericExample(cmdName, fullParentName string) string {
	tmpl := ktemplates.Examples(`  # %[1]s the %[2]s named 'foo'
  %[3]s foo`)
	return fmt.Sprintf(tmpl, strings.Title(cmdName), o.ResourceType, CommandName(cmdName, fullParentName))
}

type HalkyonEntity interface {
	Get(string, v1.GetOptions) error
	Delete(string, *v1.DeleteOptions) error
	GetKnown() []string
	GetNamespace() string
}
