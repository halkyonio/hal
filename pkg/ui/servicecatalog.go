package ui

import (
	"encoding/json"
	"fmt"
	scv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/mgutz/ansi"
	"github.com/snowdrop/odo-scaffold-plugin/pkg/validation"
	terminal2 "golang.org/x/crypto/ssh/terminal"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"sort"
	"strings"
)

// Retrieve the list of existing service class categories
func getServiceClassesCategories(categories map[string][]scv1beta1.ClusterServiceClass) (keys []string) {
	keys = make([]string, len(categories))

	i := 0
	for k := range categories {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	return keys
}

// GetServicePlanNames returns the service plan names included in the specified map
func GetServicePlanNames(stringMap map[string]scv1beta1.ClusterServicePlan) (keys []string) {
	keys = make([]string, len(stringMap))

	i := 0
	for k := range stringMap {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	return keys
}

// getServiceClassMap converts the specified array of service classes to a name-service class map
func getServiceClassMap(classes []scv1beta1.ClusterServiceClass) (classMap map[string]scv1beta1.ClusterServiceClass) {
	classMap = make(map[string]scv1beta1.ClusterServiceClass, len(classes))
	for _, v := range classes {
		classMap[v.Spec.ExternalName] = v
	}

	return classMap
}

// getServiceClassNames retrieves the keys (service class names) of the specified name-service class mappings
func getServiceClassNames(stringMap map[string]scv1beta1.ClusterServiceClass) (keys []string) {
	keys = make([]string, len(stringMap))

	i := 0
	for k := range stringMap {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	return keys
}

// SelectPlanNameInteractively lets the user to select the plan name from possible options, specifying which text should appear
// in the prompt
func SelectPlanNameInteractively(plans map[string]scv1beta1.ClusterServicePlan, promptText string) (plan string) {
	return Select(promptText, GetServicePlanNames(plans))
}

// EnterServiceNameInteractively lets the user enter the name of the service instance to create, defaulting to the provided
// default value and specifying both the prompt text and validation function for the name
func EnterServiceNameInteractively(defaultValue, promptText string) (serviceName string) {
	return Ask(promptText, defaultValue)
}

// SelectClassInteractively lets the user select target service class from possible options, first filtering by categories then
// by class name
func SelectClassInteractively(classesByCategory map[string][]scv1beta1.ClusterServiceClass) (class scv1beta1.ClusterServiceClass, serviceType string) {
	category := Select("Which kind of service do you wish to create", getServiceClassesCategories(classesByCategory))

	classes := getServiceClassMap(classesByCategory[category])

	// make a new displayClassInfo function available to survey templates to be able to add class information to the display
	displayClassInfo := "displayClassInfo"
	core.TemplateFuncs[displayClassInfo] = func(index int, pageEntries []string) string {
		if index >= 0 && len(pageEntries) > index {
			selected := pageEntries[index]
			class := classes[selected]
			return ansi.ColorCode("default+bu") + "Service class details" + ansi.ColorCode("reset") + ":\n" +
				classInfoItem("Name", class.GetExternalName()) +
				classInfoItem("Description", class.GetDescription()) +
				classInfoItem("Long", getLongDescription(class))
		}
		return "No matching entry"
	}
	defer delete(core.TemplateFuncs, displayClassInfo)

	// record original template and defer restoring it once done
	original := survey.SelectQuestionTemplate
	defer restoreOriginalTemplate(original)

	// add more information about the currently selected class
	survey.SelectQuestionTemplate = original + `
	{{- if not .ShowAnswer}}
	{{$classInfo:=(displayClassInfo .SelectedIndex .PageEntries)}}
	  {{- if $classInfo}}
{{$classInfo}}
	  {{- end}}
	{{- end}}`

	serviceType = Select("Which "+category+" service class should we use", getServiceClassNames(classes))

	return classes[serviceType], serviceType
}

// classInfoItem computes how a given service class information item should be displayed
func classInfoItem(name, value string) string {
	// wrap value if needed accounting for size of value "header" (its name)
	value = wrapIfNeeded(value, len(name)+3)

	if len(value) > 0 {
		// display the name using the default color, in bold and then reset style right after
		return StyledOutput(name, "default+b") + ": " + value + "\n"
	}
	return ""
}

// StyledOutput returns an ANSI color code to style the specified text accordingly, issuing a reset code when done using the
// https://github.com/mgutz/ansi#style-format format
func StyledOutput(text, style string) string {
	return ansi.ColorCode(style) + text + ansi.ColorCode("reset")
}

const defaultColumnNumberBeforeWrap = 80

// wrapIfNeeded wraps the given string taking the given prefix size into account based on the width of the terminal (or
// defaultColumnNumberBeforeWrap if terminal size cannot be determined).
func wrapIfNeeded(value string, prefixSize int) string {
	// get the width of the terminal
	width, _, err := terminal2.GetSize(0)
	if width == 0 || err != nil {
		// if for some reason we couldn't get the size use default value
		width = defaultColumnNumberBeforeWrap
	}

	// if the value length is greater than the width, wrap it
	// note that we need to account for the size of the name of the value being displayed before the value (i.e. its name)
	valueSize := len(value)
	if valueSize+prefixSize >= width {
		// look at each line of the value
		split := strings.Split(value, "\n")
		for index, line := range split {
			// for each line, trim it and split it in space-separated clusters ("words")
			line = strings.TrimSpace(line)
			words := strings.Split(line, " ")
			newLine := ""
			lineSize := 0

			for _, word := range words {
				if lineSize+len(word)+1+prefixSize < width {
					// concatenate word to the new computed line only if adding it to the line won't make it larger than acceptable
					newLine = newLine + " " + word
					lineSize = lineSize + 1 + len(word) // accumulate the line size
				} else {
					// otherwise, break the line and add the word on a new "line"
					newLine = newLine + "\n" + word
					lineSize = len(word) // reset the line size
				}
			}
			// replace the initial line with the new computed version
			split[index] = strings.TrimSpace(newLine)
		}
		// compute the new value by joining all the modified lines
		value = strings.Join(split, "\n")
	}
	return value
}

// restoreOriginalTemplate restores the original survey template once we're done with the display
func restoreOriginalTemplate(original string) {
	survey.SelectQuestionTemplate = original
}

// Convert the provided ClusterServiceClass to its UI representation
func getLongDescription(class scv1beta1.ClusterServiceClass) (longDescription string) {
	extension := class.Spec.ExternalMetadata
	if extension != nil {
		var meta map[string]interface{}
		err := json.Unmarshal(extension.Raw, &meta)
		if err != nil {
			fmt.Printf("Unable unmarshal Extension metadata for ClusterServiceClass '%v'", class.Spec.ExternalName)
		}
		if val, ok := meta["longDescription"]; ok {
			longDescription = val.(string)
		}
	}

	return
}

// EnterServicePropertiesInteractively lets the user enter the properties specified by the provided plan if not already
// specified by the passed values
func EnterServicePropertiesInteractively(svcPlan scv1beta1.ClusterServicePlan) (values map[string]string) {
	return enterServicePropertiesInteractively(svcPlan)
}

// enterServicePropertiesInteractively lets user enter the properties interactively using the specified Stdio instance (useful
// for testing purposes)
func enterServicePropertiesInteractively(svcPlan scv1beta1.ClusterServicePlan, stdio ...terminal.Stdio) (values map[string]string) {
	planDetails, _ := NewServicePlan(svcPlan)

	properties := make(map[string]ServicePlanParameter, len(planDetails.Parameters))
	for _, v := range planDetails.Parameters {
		properties[v.Name] = v
	}

	values = make(map[string]string, len(properties))

	sort.Sort(planDetails.Parameters)

	// first deal with required properties
	for _, prop := range planDetails.Parameters {
		if prop.Required {
			addValueFor(prop, values, stdio...)
			// remove property from list of properties to consider
			delete(properties, prop.Name)
		}
	}

	// finally check if we still have plan properties that have not been considered
	if len(properties) > 0 && Proceed("Provide values for non-required properties") {
		for _, prop := range properties {
			addValueFor(prop, values, stdio...)
		}
	}

	return values
}

func addValueFor(prop ServicePlanParameter, values map[string]string, stdio ...terminal.Stdio) {
	var result string
	prompt := &survey.Input{
		Message: fmt.Sprintf("Enter a value for %s property %s:", prop.Type, propDesc(prop)),
	}

	if len(stdio) == 1 {
		prompt.WithStdio(stdio[0])
	}

	if len(prop.Default) > 0 {
		prompt.Default = prop.Default
	}

	err := survey.AskOne(prompt, &result, GetValidatorFor(prop.AsValidatable()))
	HandleError(err)
	values[prop.Name] = result
}

// propDesc computes a human-readable description of the specified property
func propDesc(prop ServicePlanParameter) string {
	msg := ""
	if len(prop.Title) > 0 {
		msg = prop.Title
	} else if len(prop.Description) > 0 {
		msg = prop.Description
	}

	if len(msg) > 0 {
		msg = " (" + strings.TrimSpace(msg) + ")"
	}

	return prop.Name + msg
}

type servicePlanParameters []ServicePlanParameter

func (params servicePlanParameters) Len() int {
	return len(params)
}

func (params servicePlanParameters) Less(i, j int) bool {
	return params[i].Name < params[j].Name
}

func (params servicePlanParameters) Swap(i, j int) {
	params[i], params[j] = params[j], params[i]
}

// ServicePlan holds the information about service catalog plans associated to service classes
type ServicePlan struct {
	Name        string
	DisplayName string
	Description string
	Parameters  servicePlanParameters
}

// ServicePlanParameter holds the information regarding a service catalog plan parameter
type ServicePlanParameter struct {
	Name                   string `json:"name"`
	Title                  string `json:"title,omitempty"`
	Description            string `json:"description,omitempty"`
	Default                string `json:"default,omitempty"`
	validation.Validatable `json:",inline,omitempty"`
}

// NewServicePlanParameter creates a new ServicePlanParameter instance with the specified state
func NewServicePlanParameter(name, typeName, defaultValue string, required bool) ServicePlanParameter {
	return ServicePlanParameter{
		Name:    name,
		Default: defaultValue,
		Validatable: validation.Validatable{
			Type:     typeName,
			Required: required,
		},
	}
}

type serviceInstanceCreateParameterSchema struct {
	Required   []string
	Properties map[string]ServicePlanParameter
}

// NewServicePlan creates a new ServicePlan based on the specified ClusterServicePlan
func NewServicePlan(result scv1beta1.ClusterServicePlan) (plan ServicePlan, err error) {
	plan = ServicePlan{
		Name:        result.Spec.ExternalName,
		Description: result.Spec.Description,
	}

	// get the display name from the external meta data
	var externalMetaData map[string]interface{}
	err = json.Unmarshal(result.Spec.ExternalMetadata.Raw, &externalMetaData)
	if err != nil {
		return plan, err
	}

	if val, ok := externalMetaData["displayName"]; ok {
		plan.DisplayName = val.(string)
	}

	// get the create parameters
	schema := serviceInstanceCreateParameterSchema{}
	paramBytes := result.Spec.InstanceCreateParameterSchema.Raw
	err = json.Unmarshal(paramBytes, &schema)
	if err != nil {
		return plan, err
	}

	plan.Parameters = make([]ServicePlanParameter, 0, len(schema.Properties))
	for k, v := range schema.Properties {
		v.Name = k
		// we set the Required flag if the name of parameter
		// is one of the parameters indicated as required
		// these parameters are not strictly required since they might have default values
		v.Required = isRequired(schema.Required, k)

		plan.Parameters = append(plan.Parameters, v)
	}

	return
}

// isRequired checks whether the parameter with the specified name is among the given list of required ones
func isRequired(required []string, name string) bool {
	for _, n := range required {
		if n == name {
			return true
		}
	}
	return false
}
