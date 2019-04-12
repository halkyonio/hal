package ui

import (
	"fmt"
	"github.com/mgutz/ansi"
	"github.com/snowdrop/odo-scaffold-plugin/pkg/validation"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"os"
	"sort"
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
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	if len(defaultValue) == 1 {
		prompt.Default = defaultValue[0]
	}
	return askOne(prompt)
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

func Ask(message string, defaultValue ...string) string {
	input := &survey.Input{
		Message: message,
	}

	if len(defaultValue) == 1 {
		input.Default = defaultValue[0]
	}

	return askOne(input)
}

func askOne(prompt survey.Prompt, stdio ...terminal.Stdio) string {
	var response string

	err := survey.AskOne(prompt, &response, survey.Required)
	HandleError(err)

	return response
}

// GetValidatorFor returns an implementation specific validator for the given validatable to avoid type casting at each calling
// site
func GetValidatorFor(prop validation.Validatable) survey.Validator {
	return survey.Validator(validation.GetValidatorFor(prop))
}

func OutputSelection(msg, choice string) {
	fmt.Println(ansi.Green + ansi.ColorCode("default+hb") + msg + ": " + ansi.Cyan + choice + ansi.Reset)
}

func ErrorMessage(msg, wrong string) string {
	return fmt.Sprintf("%s%s: %s%s\nSelect other(s) from:", ansi.Red, msg, wrong, ansi.ColorCode("default"))
}
