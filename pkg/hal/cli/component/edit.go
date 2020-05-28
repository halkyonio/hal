package component

import (
	"fmt"
	"github.com/spf13/cobra"
	v1beta13 "halkyon.io/api/capability/v1beta1"
	"halkyon.io/api/component/v1beta1"
	v1beta12 "halkyon.io/api/runtime/v1beta1"
	halkyon "halkyon.io/api/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/hal/cli/capability"
	"halkyon.io/hal/pkg/ui"
	"halkyon.io/hal/pkg/validation"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"strings"
)

type editOptions struct {
	*cmdutil.CreateOptions
	*cmdutil.EnvOptions
	*v1beta12.GeneratorOptions
	runtime      string
	exposeP      string
	expose       bool
	port         int
	requiredCaps []v1beta1.RequiredCapabilityConfig
	providedCaps []v1beta1.CapabilityConfig
	target       *v1beta1.Component
}

func (o *editOptions) GeneratePrefix() string {
	return o.runtime
}

func (o *editOptions) Build() runtime.Object {
	return o.target
}

func (o *editOptions) Set(entity runtime.Object) {
	o.target = entity.(*v1beta1.Component)
	o.runtime = o.target.Spec.Runtime
	o.RuntimeVersion = o.target.Spec.Version
	o.port = int(o.target.Spec.Port)
	o.requiredCaps = o.target.Spec.Capabilities.Requires
	o.providedCaps = o.target.Spec.Capabilities.Provides
}

func (o *editOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	ui.SelectOrCheckExisting(&o.runtime, "Runtime", o.getRuntimes(), o.isValidRuntime)
	ui.SelectOrCheckExisting(&o.RuntimeVersion, "Version", o.getVersionsForRuntime(), o.isValidVersionGivenRuntime)

	if ui.Proceed("Edit required capabilities") {
		for {
			done, remove, disp := displayAndControl("required", func() ui.DisplayableMap {
				result := ui.NewDisplayableMap(len(o.target.Spec.Capabilities.Requires))
				for i, required := range o.target.Spec.Capabilities.Requires {
					result.Add(displayable{element: required, index: i})
				}
				return result
			})
			if done {
				break
			}
			if remove && disp != nil {
				rd := disp.(displayable)
				o.target.Spec.Capabilities.Requires = append(o.target.Spec.Capabilities.Requires[:rd.index], o.target.Spec.Capabilities.Requires[rd.index+1:]...)
				continue
			}
			required := v1beta1.RequiredCapabilityConfig{}
			if add != disp.Name() {
				required = disp.GetUnderlying().(v1beta1.RequiredCapabilityConfig)
			}
			required.Name = ui.Ask("Name", "", required.Name)

			existing := capability.Entity.GetMatching()
			hasCaps := existing.Len() > 0
			if hasCaps && ui.Proceed("Bind to existing capability") {
				displayable := ui.SelectDisplayable("Target capability", existing)
				required.BoundTo = displayable.Name()
				required.Spec = displayable.GetUnderlying().(v1beta13.Capability).Spec
			} else {
				capCreate := capability.CapabilityCreateOptions{}
				if err := capCreate.Complete(); err != nil {
					return err
				}
				required.Spec = capCreate.AsCapabilitySpec()
				required.AutoBindable = ui.Proceed("Auto-bindable")
			}
			if ui.Proceed("Add extra parameters") {
				for {
					paramPair := ui.AskOrReturnToExit("Parameter in the 'name=value' format, simply press enter when finished")
					if len(paramPair) == 0 {
						break
					}
					split := strings.Split(paramPair, "=")
					if len(split) != 2 {
						return fmt.Errorf("invalid parameter: %s, format must be 'name=value'", paramPair)
					}
					param := halkyon.NameValuePair{Name: split[0], Value: split[1]}
					required.Spec.Parameters = append(required.Spec.Parameters, param)
					ui.OutputSelection("Set parameter", fmt.Sprintf("%s=%s", param.Name, param.Value))
				}
			}
			if rd, ok := disp.(displayable); ok {
				o.target.Spec.Capabilities.Requires[rd.index] = required
			} else {
				o.target.Spec.Capabilities.Requires = append(o.target.Spec.Capabilities.Requires, required)
			}
		}
	}

	if ui.Proceed("Edit provides capabilities") {
		for {
			done, remove, disp := displayAndControl("provided", func() ui.DisplayableMap {
				result := ui.NewDisplayableMap(len(o.target.Spec.Capabilities.Provides))
				for i, required := range o.target.Spec.Capabilities.Provides {
					result.Add(displayable{element: required, index: i})
				}
				return result
			})
			if done {
				break
			}
			if remove && disp != nil {
				rd := disp.(displayable)
				o.target.Spec.Capabilities.Provides = append(o.target.Spec.Capabilities.Provides[:rd.index], o.target.Spec.Capabilities.Provides[rd.index+1:]...)
				continue
			}
			provided := v1beta1.CapabilityConfig{}
			if add != disp.Name() {
				provided = disp.GetUnderlying().(v1beta1.CapabilityConfig)
			}
			provided.Name = ui.Ask("Name", "", provided.Name)
			capCreate := capability.CapabilityCreateOptions{}
			if err := capCreate.Complete(); err != nil {
				return err
			}
			if err := capCreate.Validate(); err != nil {
				return err
			}
			provided.Spec = capCreate.AsCapabilitySpec()
			if rd, ok := disp.(displayable); ok {
				o.target.Spec.Capabilities.Provides[rd.index] = provided
			} else {
				o.target.Spec.Capabilities.Provides = append(o.target.Spec.Capabilities.Provides, provided)
			}
		}
	}

	if err := o.EnvOptions.Complete(name, cmd, args); err != nil {
		return err
	}

	return nil
}

const add = "__add__"
const done = "__done__"
const remove = "__remove__"

func displayAndControl(capType string, displayableMap func() ui.DisplayableMap) (bool, bool, ui.Displayable) {
	result := displayableMap()
	if result.Len() > 0 {
		result.Add(ui.NewControlDisplayable(remove, ui.ControlString(fmt.Sprintf("- Remove %s capability", capType))))
	}
	result.Add(ui.NewControlDisplayable(add, ui.ControlString(fmt.Sprintf("+ Add new %s capability", capType))))
	result.Add(ui.NewControlDisplayable(done, ui.ControlString("✓ Done (select to exit)")))
	displayable := ui.SelectDisplayable(fmt.Sprintf("Select or add %s capabilities", capType), result)

	if done == displayable.Name() {
		return true, false, nil
	}

	if remove == displayable.Name() {
		result := displayableMap()
		d := ui.SelectDisplayable(fmt.Sprintf("Select %s capability to remove", capType), result)
		if ui.Proceed("Really remove") {
			return false, true, d
		}
		return false, true, nil
	}
	return false, false, displayable
}

func (o *editOptions) Validate() error {
	currentDir, _ := os.Getwd()
	children := o.getChildDirNames()
	if !validation.IsValidDir(o.Name) {
		if len(children) == 0 || ui.Proceed(fmt.Sprintf("no directory named '%s' exists in %v, create it", o.Name, currentDir)) {
			// if we're not scaffolding and we don't have any existing children directory, create one
			err := os.Mkdir(o.Name, os.ModePerm)
			if err != nil {
				return err
			}
			ui.OutputSelection("Created new component directory", o.Name)
		} else {
			return fmt.Errorf("'%s' directory was not created in %v", o.Name, currentDir)
		}
	}
	return nil
}

func (o *editOptions) getRuntimes() []string {
	return getRuntimeNames()
}

func (o *editOptions) isValidRuntime() bool {
	return validation.IsValid(o.runtime, o.getRuntimes())
}

func (o *editOptions) getVersionsForRuntime() []string {
	r, ok := runtimes[o.runtime]
	if !ok {
		return []string{"Unknown runtime " + o.runtime} // shouldn't happen
	}
	return r.versions
}

func (o *editOptions) isValidVersionGivenRuntime() bool {
	return validation.IsValid(o.RuntimeVersion, o.getVersionsForRuntime())
}

func (o *editOptions) getChildDirNames() []string {
	currentDir, _ := os.Getwd()
	childDirs := make([]string, 0, 7)
	children, err := ioutil.ReadDir(currentDir)
	if err != nil {
		panic(err)
	}
	for _, child := range children {
		if child.IsDir() {
			name := child.Name()
			if !strings.HasPrefix(name, ".") {
				childDirs = append(childDirs, name)
			}
		}
	}
	return childDirs
}

func (o *editOptions) SetEnvOptions(env *cmdutil.EnvOptions) {
	o.EnvOptions = env
}

func NewCmdEdit(fullParentName string) *cobra.Command {
	o := &editOptions{}
	generic := cmdutil.NewCreateOptions(cmdutil.Component, Entity)
	generic.Delegate = o
	o.CreateOptions = generic
	o.CreateOptions.OperationName = "edit"
	o.GeneratorOptions = &v1beta12.GeneratorOptions{}
	cmd := cmdutil.NewGenericCreate(fullParentName, generic)
	cmd.Example = fmt.Sprintf(createExample, cmdutil.CommandName(cmd.Name(), fullParentName))

	cmd.Flags().StringVarP(&o.runtime, "runtime", "r", "", "Runtime to use for the component. Possible values:"+strings.Join(getRuntimeNames(), ","))
	cmd.Flags().StringVarP(&o.RuntimeVersion, "runtimeVersion", "i", "", "Runtime version")
	cmd.Flags().StringVarP(&o.exposeP, "expose", "x", "", "Whether or not to expose the microservice outside of the cluster")
	cmd.Flags().IntVarP(&o.port, "port", "o", 0, "Port the microservice listens on")

	cmdutil.SetupEnvOptions(o, cmd)

	return cmd
}

type displayable struct {
	element interface{}
	index   int
}

func (rd displayable) Help() string {
	return rd.Display()
}

func (rd displayable) getConfig() v1beta1.CapabilityConfig {
	switch v := rd.element.(type) {
	case v1beta1.CapabilityConfig:
		return v
	case v1beta1.RequiredCapabilityConfig:
		return v.CapabilityConfig
	default:
		panic(fmt.Sprintf("unexpected element type %T!", v))
	}
}

func (rd displayable) Display() string {
	return capability.GetDisplay(rd.Name(), rd.getConfig().Spec)
}

func (rd displayable) Name() string {
	return rd.getConfig().Name
}

func (rd displayable) GetUnderlying() interface{} {
	return rd.element
}
