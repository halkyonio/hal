package capability

import (
	"fmt"
	"github.com/spf13/cobra"
	v1beta12 "halkyon.io/api/capability-info/v1beta1"
	"halkyon.io/api/capability/v1beta1"
	halkyon "halkyon.io/api/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/ui"
	"halkyon.io/hal/pkg/validation"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"sort"
	"strings"
)

type typeRegistry map[string][]string
type categoryRegisty map[string]typeRegistry

var categories = <-getCapabilityInfos()

type CapabilityCreateOptions struct {
	category    string
	subCategory string
	version     string
	paramPairs  []string
	parameters  []halkyon.NameValuePair
}

func (c CapabilityCreateOptions) AsCapabilitySpec() v1beta1.CapabilitySpec {
	return v1beta1.CapabilitySpec{
		Category:   v1beta1.CapabilityCategory(c.category),
		Type:       v1beta1.CapabilityType(c.subCategory),
		Version:    c.version,
		Parameters: c.parameters,
	}
}

type createOptions struct {
	CapabilityCreateOptions
	*cmdutil.CreateOptions
	target *v1beta1.Capability
}

func (o *createOptions) Set(entity runtime.Object) {
	o.target = entity.(*v1beta1.Capability)
}

var (
	capabilityExample = ktemplates.Examples(`  # Create a new database capability of type postgres 10 and sets up some parameters as the name of the database and the user/password to connect.
  %[1]s -n db-capability -g database -t postgres -v 10 -p DB_NAME=sample-db -p DB_PASSWORD=admin -p DB_USER=admin`)
)

func (o *createOptions) GeneratePrefix() string {
	return o.subCategory
}

func (o *createOptions) Build() runtime.Object {
	if o.target == nil {
		o.target = &v1beta1.Capability{
			ObjectMeta: v1.ObjectMeta{
				Name:      o.Name,
				Namespace: o.CreateOptions.Client.GetNamespace(),
			},
			Spec: o.AsCapabilitySpec(),
		}
	}
	return o.target
}

func (o *createOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	return o.CapabilityCreateOptions.Complete()
}

func (o *createOptions) Validate() error {
	return o.CapabilityCreateOptions.Validate()
}

func (c *CapabilityCreateOptions) Complete() error {
	ui.SelectOrCheckExisting(&c.category, "Category", c.getCategories(), c.isValidCategory)
	ui.SelectOrCheckExisting(&c.subCategory, "Type", c.getTypesFor(c.category), c.isValidTypeGivenCategory)
	ui.SelectOrCheckExisting(&c.version, "Version", c.getVersionsFor(c.category, c.subCategory), c.isValidVersionGivenCategoryAndType)

	for _, pair := range c.paramPairs {
		if e := c.addToParams(pair); e != nil {
			return e
		}
	}

	return nil
}

func (c *CapabilityCreateOptions) Validate() error {
	infos := c.getParameterInfos()

	params := make(map[string]parameterInfo, len(infos))
	for _, v := range infos {
		params[v.name] = v
	}

	if len(c.parameters) == 0 {
		c.parameters = make([]halkyon.NameValuePair, 0, len(params))
	}

	// first deal with required params
	for _, info := range infos {
		if info.Required {
			c.addValueFor(info)
			// remove property from list of properties to consider
			delete(params, info.name)
		}
	}

	// finally check if we still have capability parameters that have not been considered
	if len(params) > 0 && ui.Proceed("Provide values for non-required parameters") {
		for _, prop := range params {
			c.addValueFor(prop)
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

func (c *CapabilityCreateOptions) getCategories() []string {
	result := make([]string, 0, len(categories))
	for k := range categories {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

func (c *CapabilityCreateOptions) isValidCategory() bool {
	return validation.IsValid(c.category, c.getCategories())
}

func (c *CapabilityCreateOptions) getTypesFor(category string) []string {
	types := categories[category]
	result := make([]string, 0, len(types))
	for k := range types {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

func (c *CapabilityCreateOptions) isValidTypeGivenCategory() bool {
	return c.isValidTypeFor(c.category)
}

func (c *CapabilityCreateOptions) isValidTypeFor(category string) bool {
	return validation.IsValid(c.subCategory, c.getTypesFor(category))
}

func (c *CapabilityCreateOptions) getVersionsFor(category, subCategory string) []string {
	return categories[category][subCategory]
}

func (c *CapabilityCreateOptions) isValidVersionFor(category, subCategory string) bool {
	return validation.IsValid(c.version, c.getVersionsFor(category, subCategory))
}

func (c *CapabilityCreateOptions) isValidVersionGivenCategoryAndType() bool {
	return c.isValidVersionFor(c.category, c.subCategory)
}

func (c *CapabilityCreateOptions) addToParams(pair string) error {
	// todo: extract as generic version to be used for Envs and Parameters
	split := strings.Split(pair, "=")
	if len(split) != 2 {
		return fmt.Errorf("invalid parameter: %s, format must be 'name=value'", pair)
	}
	parameter := halkyon.NameValuePair{Name: split[0], Value: split[1]}
	c.parameters = append(c.parameters, parameter)
	return nil
}

func (c *CapabilityCreateOptions) getParameterInfos() []parameterInfo {
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

func (c *CapabilityCreateOptions) addValueFor(prop parameterInfo) {
	// first look if we have provided a value for this already
	provided := ""
	for _, parameter := range c.parameters {
		if parameter.Name == prop.name {
			provided = parameter.Value
		}
	}
	result := ui.Ask(fmt.Sprintf("Value for %s property %s:", prop.Type, prop.name), provided)
	if result != provided {
		c.parameters = append(c.parameters, halkyon.NameValuePair{
			Name:  prop.name,
			Value: result,
		})
	}
}

func getCapabilityInfos() chan categoryRegisty {
	r := make(chan categoryRegisty)

	go func() {
		list, err := k8s.GetClient().HalkyonCapabilityInfoClient.CapabilityInfos().List(v1.ListOptions{})
		if err != nil {
			panic(err)
		}

		capInfos := make(categoryRegisty, 11)
		for _, item := range list.Items {
			category := item.Spec.Category
			types, ok := capInfos[category]
			if !ok {
				types = make(typeRegistry, 7)
				capInfos[category] = types
			}

			_, ok = types[item.Spec.Type]
			if !ok {
				types[item.Spec.Type] = strings.Split(item.Spec.Versions, v1beta12.CapabilityInfoVersionSeparator)
			} else {
				panic(fmt.Errorf("a type named %s is already registered for category %s", item.Spec.Type, category))
			}
		}

		r <- capInfos
	}()

	return r
}

func NewCmdCreate(parent string) *cobra.Command {
	o := &createOptions{}
	generic := cmdutil.NewCreateOptions(cmdutil.Capability, Entity)
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
