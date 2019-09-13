package component

import (
	"fmt"
	"github.com/pkg/errors"
	component "halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record/util"
)

type commonOptions struct {
	*cmdutil.ComponentTargetingOptions
}

func (o *commonOptions) SetTargetingOptions(options *cmdutil.ComponentTargetingOptions) {
	o.ComponentTargetingOptions = options
}

func (o *commonOptions) createIfNeeded() (*component.Component, error) {
	c := k8s.GetClient()
	name := o.GetTargetedComponentName()
	comp, err := c.HalkyonComponentClient.Components(c.Namespace).Get(name, v1.GetOptions{})
	if err != nil {
		// check error to see if it means that the component doesn't exist yet
		if util.IsKeyNotFoundError(errors.Cause(err)) {
			// the component was not found so we need to create it first and wait for it to be ready
			log.Infof("'%s' component was not found, initializing it", name)
			err = k8s.Apply(o.GetTargetedComponentDescriptor(), c.Namespace)
			if err != nil {
				return nil, fmt.Errorf("error applying component CR: %v", err)
			}

			return o.waitUntilReady(comp)
		} else {
			return nil, err
		}
	}
	return comp, nil
}

func (o *commonOptions) waitUntilReady(c *component.Component) (*component.Component, error) {
	if component.ComponentReady == c.Status.Phase || component.ComponentRunning == c.Status.Phase {
		return c, nil
	}

	name := o.GetTargetedComponentName()
	client := k8s.GetClient()
	cp, err := client.WaitForComponent(name, component.ComponentReady, "Waiting for component "+name+" to be ready…")
	if err != nil {
		return nil, fmt.Errorf("error waiting for component: %v", err)
	}
	err = errorIfFailedOrUnknown(c)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

func errorIfFailedOrUnknown(c *component.Component) error {
	switch c.Status.Phase {
	case component.ComponentFailed, component.ComponentUnknown:
		return errors.Errorf("status of component %s is %s: %s", c.Name, c.Status.Phase, c.Status.Message)
	default:
		return nil
	}
}
