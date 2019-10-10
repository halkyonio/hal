package capability

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/api/capability/v1beta1"
	halkyon "halkyon.io/api/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/ui"
	"halkyon.io/hal/pkg/validation"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"strings"
)

type createOptions struct {
	category    string
	subCategory string
	version     string
	paramPairs  []string
	parameters  []halkyon.NameValuePair
	*cmdutil.CreateOptions
	capability *v1beta1.Capability
}

func (o *createOptions) Set(object runtime.Object) {
	o.capability = object.(*v1beta1.Capability)
}

var (
	capabilityExample = ktemplates.Examples(`  # Create a new database capability of type postgres 10 and sets up some parameters as the name of the database and the user/password to connect.
  %[1]s -n db-capability -g database -t postgres -v 10 -p DB_NAME=sample-db -p DB_PASSWORD=admin -p DB_USER=admin`)
)

func (o *createOptions) GeneratePrefix() string {
	return o.subCategory
}

func (o *createOptions) Build() runtime.Object {
	return &v1beta1.Capability{
		ObjectMeta: v1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.CreateOptions.Client.GetNamespace(),
		},
		Spec: v1beta1.CapabilitySpec{
			Category:   v1beta1.DatabaseCategory, // todo: replace hardcoded value
			Type:       v1beta1.PostgresType,     // todo: replace hardcoded value
			Version:    o.version,
			Parameters: o.parameters,
		},
	}
}

func (o *createOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	ui.SelectOrCheckExisting(&o.category, "Category", o.getCategories(), o.isValidCategory)
	ui.SelectOrCheckExisting(&o.subCategory, "Type", o.getTypesFor(o.category), o.isValidTypeGivenCategory)
	ui.SelectOrCheckExisting(&o.version, "Version", o.getVersionsFor(o.category, o.subCategory), o.isValidVersionGivenCategoryAndType)

	for _, pair := range o.paramPairs {
		if e := o.addToParams(pair); e != nil {
			return e
		}
	}

	return nil
}

func (o *createOptions) Validate() error {
	infos := o.getParameterInfos()

	params := make(map[string]parameterInfo, len(infos))
	for _, v := range infos {
		params[v.name] = v
	}

	if len(o.parameters) == 0 {
		o.parameters = make([]halkyon.NameValuePair, 0, len(params))
	}

	// first deal with required params
	for _, info := range infos {
		if info.Required {
			o.addValueFor(info)
			// remove property from list of properties to consider
			delete(params, info.name)
		}
	}

	// finally check if we still have capability parameters that have not been considered
	if len(params) > 0 && ui.Proceed("Provide values for non-required parameters") {
		for _, prop := range params {
			o.addValueFor(prop)
		}
	}

	return nil
}

type parameterInfo struct {
	validation.Validatable
	name string
}

func (p parameterInfo) AsValidatable() validation.Validatable {
	return p.Validatable
}

func (o *createOptions) getCategories() []string {
	// todo: implement operator querying of available capabilities
	return []string{"database"}
}

func (o *createOptions) isValidCategory() bool {
	return validation.IsValid(o.category, o.getCategories())
}

func (o *createOptions) getTypesFor(category string) []string {
	// todo: implement operator querying for available types for given category
	return []string{"postgres"}
}

func (o *createOptions) isValidTypeGivenCategory() bool {
	return o.isValidTypeFor(o.category)
}

func (o *createOptions) isValidTypeFor(category string) bool {
	return validation.IsValid(o.subCategory, o.getTypesFor(category))
}

func (o *createOptions) getVersionsFor(category, subCategory string) []string {
	// todo: implement operator querying
	return []string{"11", "10", "9.6", "9.5", "9.4"}
}

func (o *createOptions) isValidVersionFor(category, subCategory string) bool {
	return validation.IsValid(o.version, o.getVersionsFor(category, subCategory))
}

func (o *createOptions) isValidVersionGivenCategoryAndType() bool {
	return o.isValidVersionFor(o.category, o.subCategory)
}

func (o *createOptions) addToParams(pair string) error {
	// todo: extract as generic version to be used for Envs and Parameters
	split := strings.Split(pair, "=")
	if len(split) != 2 {
		return fmt.Errorf("invalid parameter: %s, format must be 'name=value'", pair)
	}
	parameter := halkyon.NameValuePair{Name: split[0], Value: split[1]}
	o.parameters = append(o.parameters, parameter)
	return nil
}

func (o *createOptions) getParameterInfos() []parameterInfo {
	// todo: implement operator querying
	infos := make([]parameterInfo, 3, 3)
	infos[0] = parameterInfo{
		Validatable: validation.Validatable{
			Required: true,
			Type:     "string",
		},
		name: "DB_NAME",
	}
	infos[1] = parameterInfo{
		name: "DB_PASSWORD",
		Validatable: validation.Validatable{
			Required: true,
			Type:     "string",
		},
	}
	infos[2] = parameterInfo{
		name: "DB_USER",
		Validatable: validation.Validatable{
			Required: true,
			Type:     "string",
		},
	}
	return infos
}

func (o *createOptions) addValueFor(prop parameterInfo) {
	// first look if we have provided a value for this already
	provided := ""
	for _, parameter := range o.parameters {
		if parameter.Name == prop.name {
			provided = parameter.Value
		}
	}
	result := ui.Ask(fmt.Sprintf("Value for %s property %s:", prop.Type, prop.name), provided)
	if result != provided {
		o.parameters = append(o.parameters, halkyon.NameValuePair{
			Name:  prop.name,
			Value: result,
		})
	}
}

func NewCmdCreate(parent string) *cobra.Command {
	c := k8s.GetClient()
	o := &createOptions{}
	generic := cmdutil.NewCreateOptions("capability", client{
		client: c.HalkyonCapabilityClient.Capabilities(c.Namespace),
		ns:     c.Namespace,
	})
	generic.Delegate = o
	o.CreateOptions = generic
	capability := cmdutil.NewGenericCreate(parent, generic)
	capability.Example = fmt.Sprintf(capabilityExample, cmdutil.CommandName(capability.Name(), parent))

	capability.Flags().StringVarP(&o.category, "category", "g", "", "Capability category e.g. 'database'")
	capability.Flags().StringVarP(&o.subCategory, "type", "t", "", "Capability type e.g. 'postgres'")
	capability.Flags().StringVarP(&o.version, "version", "v", "", "Capability version")
	capability.Flags().StringSliceVarP(&o.paramPairs, "parameters", "p", []string{}, "Capability-specific parameters")

	return capability
}
