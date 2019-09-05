package capability

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"halkyon.io/api/capability/v1beta1"
	halkyon "halkyon.io/api/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
	"halkyon.io/hal/pkg/validation"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

const capabilityCommandName = "capability"

type capabilityOptions struct {
	category    string
	subCategory string
	version     string
	paramPairs  []string
	parameters  []halkyon.NameValuePair

	name string
}

func (o *capabilityOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	o.selectOrCheckExisting(&o.category, "Category", o.getCategories(), o.isValidCategory)
	o.selectOrCheckExisting(&o.subCategory, "Type", o.getTypesFor(o.category), o.isValidTypeGivenCategory)
	o.selectOrCheckExisting(&o.version, "Version", o.getVersionsFor(o.category, o.subCategory), o.isValidVersionGivenCategoryAndType)

	for _, pair := range o.paramPairs {
		if e := o.addToParams(pair); e != nil {
			return e
		}
	}

	generated := fmt.Sprintf("%s-capability-%d", o.subCategory, time.Now().UnixNano())
	o.name = ui.Ask("Change default name", o.name, generated)

	return nil
}

func (o *capabilityOptions) Validate() error {
	infos := o.getParameterInfos()

	params := make(map[string]parameterInfo, len(infos))
	for _, v := range infos {
		params[v.name] = v
	}

	o.parameters = make([]halkyon.NameValuePair, len(params))

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

func (o *capabilityOptions) Run() error {
	client := k8s.GetClient()
	c, err := client.HalkyonCapabilityClient.Capabilities(client.Namespace).Create(&v1beta1.Capability{
		ObjectMeta: v1.ObjectMeta{
			Name:      o.name,
			Namespace: client.Namespace,
		},
		Spec: v1beta1.CapabilitySpec{
			Category:   v1beta1.DatabaseCategory, // todo: replace hardcoded value
			Type:       v1beta1.PostgresType,     // todo: replace hardcoded value
			Version:    o.version,
			Parameters: o.parameters,
		},
	})

	if err != nil {
		return err
	}

	log.Successf("Created capability %s", c.Name)

	return nil
}

func (o *capabilityOptions) selectOrCheckExisting(parameterValue *string, capitalizedParameterName string, validValues []string, validator func() bool) {
	if len(*parameterValue) == 0 {
		*parameterValue = ui.Select(capitalizedParameterName, validValues)
	} else {
		lowerCaseParameterName := strings.ToLower(capitalizedParameterName)
		if !validator() {
			s := ui.ErrorMessage("Unknown "+lowerCaseParameterName, *parameterValue)
			ui.Select(s, validValues)
		} else {
			ui.OutputSelection("Selected "+lowerCaseParameterName, *parameterValue)
		}
	}
}

func (o *capabilityOptions) getCategories() []string {
	// todo: implement operator querying of available capabilities
	return []string{"database"}
}

func (o *capabilityOptions) isValidCategory() bool {
	return isValid(o.category, o.getCategories())
}

func isValid(value string, validValues []string) bool {
	for _, v := range validValues {
		if value == v {
			return true
		}
	}
	return false
}

func (o *capabilityOptions) getTypesFor(category string) []string {
	// todo: implement operator querying for available types for given category
	return []string{"postgres"}
}

func (o *capabilityOptions) isValidTypeGivenCategory() bool {
	return o.isValidTypeFor(o.category)
}

func (o *capabilityOptions) isValidTypeFor(category string) bool {
	return isValid(o.subCategory, o.getTypesFor(category))
}

func (o *capabilityOptions) getVersionsFor(category, subCategory string) []string {
	// todo: implement operator querying
	return []string{"11", "10", "9.6", "9.5", "9.4"}
}

func (o *capabilityOptions) isValidVersionFor(category, subCategory string) bool {
	return isValid(o.version, o.getVersionsFor(category, subCategory))
}

func (o *capabilityOptions) isValidVersionGivenCategoryAndType() bool {
	return o.isValidVersionFor(o.category, o.subCategory)
}

func (o *capabilityOptions) addToParams(pair string) error {
	// todo: extract as generic version to be used for Envs and Parameters
	split := strings.Split(pair, "=")
	if len(split) != 2 {
		return fmt.Errorf("invalid parameter: %s, format must be 'name=value'", pair)
	}
	parameter := halkyon.NameValuePair{Name: split[0], Value: split[1]}
	o.parameters = append(o.parameters, parameter)
	ui.OutputSelection("Set parameter", fmt.Sprintf("%s=%s", parameter.Name, parameter.Value))
	return nil
}

func (o *capabilityOptions) getParameterInfos() []parameterInfo {
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

func (o *capabilityOptions) addValueFor(prop parameterInfo) {
	var result string
	prompt := &survey.Input{
		Message: fmt.Sprintf("Enter a value for %s property %s:", prop.Type, prop.name),
	}

	err := survey.AskOne(prompt, &result, ui.GetValidatorFor(prop.AsValidatable()))
	ui.HandleError(err)
	o.parameters = append(o.parameters, halkyon.NameValuePair{
		Name:  prop.name,
		Value: result,
	})
}

func NewCmdCapability(parent string) *cobra.Command {
	o := &capabilityOptions{}
	capability := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", capabilityCommandName),
		Short: "Create a new capability",
		Long:  `Create a new capability`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.GenericRun(o, cmd, args)
		},
	}

	capability.Flags().StringVarP(&o.category, "category", "g", "", "Capability category e.g. 'database'")
	capability.Flags().StringVarP(&o.subCategory, "type", "t", "", "Capability type e.g. 'postgres'")
	capability.Flags().StringVarP(&o.version, "version", "v", "", "Capability version")
	capability.Flags().StringSliceVarP(&o.paramPairs, "parameters", "p", []string{}, "Capability-specific parameters")

	return capability
}
