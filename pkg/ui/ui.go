package ui

import (
	"fmt"
	"github.com/mgutz/ansi"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"halkyon.io/hal/pkg/validation"
	"os"
	"sort"
	"strings"
)

// HandleError handles UI-related errors, in particular useful to gracefully handle ctrl-c interrupts gracefully
func HandleError(err error) {
	if err != nil {
		if err == terminal.InterruptErr {
			os.Exit(1)
		} else {
			fmt.Printf("Encountered an error processing prompt: %v", err)
		}
	}
}

// Proceed displays a given message and asks the user if they want to proceed
func Proceed(message string) bool {
	var response bool
	prompt := &survey.Confirm{
		Message: message,
		Default: true,
	}

	err := survey.AskOne(prompt, &response, survey.Required)
	HandleError(err)

	return response
}

func Select(message string, options []string, defaultValue ...string) string {
	sort.Strings(options)
	return doSelect(message, options, defaultValue)
}

func SelectUnsorted(message string, options []string, defaultValue ...string) string {
	return doSelect(message, options, defaultValue)
}

func doSelect(message string, options []string, defaultValue []string) string {
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	if len(defaultValue) == 1 {
		prompt.Default = defaultValue[0]
	}
	return askOne(prompt, survey.Required)
}

func MultiSelect(message string, options []string, defaultValues []string) []string {
	sort.Strings(options)
	modules := []string{}
	prompt := &survey.MultiSelect{
		Message: message,
		Options: options,
		Default: defaultValues,
	}
	err := survey.AskOne(prompt, &modules, survey.Required)
	HandleError(err)
	return modules
}

func AskOrReturnToExit(message string, defaultValue ...string) string {
	input := &survey.Input{
		Message: message,
	}

	if len(defaultValue) == 1 {
		input.Default = defaultValue[0]
	}

	return askOne(input, validation.NilValidator)
}

func Ask(message, provided string, defaultValue ...string) string {
	input := &survey.Input{
		Message: message,
	}

	if len(defaultValue) == 1 {
		input.Default = defaultValue[0]
	}

	if len(provided) > 0 && provided != "0" {
		// todo: validate provided and ask if value is invalid
		OutputSelection("Selected "+message, provided)
		return provided
	}
	return askOne(input, survey.Required)
}

func askOne(prompt survey.Prompt, validator survey.Validator, stdio ...terminal.Stdio) string {
	var response string

	err := survey.AskOne(prompt, &response, validator)
	HandleError(err)

	return response
}

// GetValidatorFor returns an implementation specific validator for the given validatable to avoid type casting at each calling
// site
func GetValidatorFor(prop validation.Validatable) survey.Validator {
	return survey.Validator(validation.GetValidatorFor(prop))
}

func OutputSelection(msg, choice string) {
	fmt.Println(ansi.Green + ansi.ColorCode("default+hb") + core.SelectFocusIcon + " " + msg + ": " + ansi.Cyan + choice + ansi.Reset)
}

func OutputError(msg string) {
	fmt.Println(ansi.Red + ansi.ColorCode("default+hb") + core.ErrorIcon + " " + msg + ansi.Reset)
}

func OutputMessage(msg string) {
	fmt.Println(ansi.Cyan + ansi.ColorCode("default+hb") + msg + ansi.Reset)
}

func SelectFromOtherErrorMessage(msg, wrong string) string {
	return fmt.Sprintf("%s%s: %s%s\nSelect other(s) from:", ansi.Red, msg, wrong, ansi.ColorCode("default"))
}

func SelectOrCheckExisting(parameterValue *string, capitalizedParameterName string, validValues []string, validator func() bool) {
	lowerCaseParameterName := strings.ToLower(capitalizedParameterName)
	if len(*parameterValue) == 0 {
		if len(validValues) == 1 {
			*parameterValue = validValues[0]
			OutputSelection("Automatically selected only available "+lowerCaseParameterName, *parameterValue)
			return
		}
		*parameterValue = Select(capitalizedParameterName, validValues)
	} else {
		if !validator() {
			s := SelectFromOtherErrorMessage("Unknown "+lowerCaseParameterName, *parameterValue)
			Select(s, validValues)
		} else {
			OutputSelection("Selected "+lowerCaseParameterName, *parameterValue)
		}
	}
}

func init() {
	core.SetFancyIcons()
}
