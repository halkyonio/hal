package validation

import (
	"fmt"
	"gopkg.in/AlecAivazis/survey.v1"
	"strconv"
	"strings"
)

// NameValidator provides a Validator view of the ValidateName function.
func NameValidator(name interface{}) error {
	if s, ok := name.(string); ok {
		return ValidateName(s)
	}

	return fmt.Errorf("can only validate strings, got %v", name)
}

// Validator is a function that validates that the provided interface conforms to expectations or return an error
type Validator func(interface{}) error

// NilValidator always validates
func NilValidator(interface{}) error { return nil }

// IntegerValidator validates that the provided object can be properly converted to an int value
func IntegerValidator(ans interface{}) error {
	if _, ok := ans.(int); ok {
		return nil
	}

	if s, ok := ans.(string); ok {
		_, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid integer value '%s': %s", s, err)
		}
		return nil
	}

	return fmt.Errorf("don't know how to convert %v into an integer", ans)
}

type StringerSet struct {
	values      map[string]bool
	valueName   string
	knownValues string
}

func NewStringerSet(valueName string, values ...interface{}) StringerSet {
	if len(values) > 0 {
		valueMap := make(map[string]bool, len(values))
		for _, value := range values {
			if s, ok := value.(fmt.Stringer); ok {
				valueMap[s.String()] = true
			}
		}
		set := StringerSet{
			values:    valueMap,
			valueName: valueName,
		}
		set.GetKnownValues() // initialize known values
		return set
	}
	panic(fmt.Errorf("a StringerSet must contain at least one possible value"))
}
func (set StringerSet) Contains(ans interface{}) error {
	if s, ok := ans.(fmt.Stringer); ok {
		if !set.values[s.String()] {
			return fmt.Errorf("unknown %s: %s, valid %ss are: %s", set.valueName, s, set.valueName, set.knownValues)
		}
	}
	return fmt.Errorf("can only validate Stringer instances, was given: %v", ans)
}
func (set StringerSet) GetKnownValues() string {
	if len(set.knownValues) == 0 {
		values := make([]string, 0, len(set.values))
		for value := range set.values {
			values = append(values, value)
		}
		set.knownValues = strings.Join(values, ",")
	}
	return set.knownValues
}

// GetValidatorFor retrieves a validator for the specified validatable, first validating its required state, then its value
// based on type then any additional validators in the order specified by Validatable.AdditionalValidators
func GetValidatorFor(prop Validatable) Validator {
	v, _ := internalGetValidatorFor(prop)
	return v
}

// internalGetValidatorFor exposed for testing purposes
func internalGetValidatorFor(prop Validatable) (validator Validator, chain []survey.Validator) {
	// make sure we don't run into issues when composing validators
	validatorChain := make([]survey.Validator, 0, 5)

	if prop.Required {
		validatorChain = append(validatorChain, survey.Required)
	}

	switch prop.Type {
	case "integer":
		validatorChain = append(validatorChain, IntegerValidator)
	}

	for i := range prop.AdditionalValidators {
		validatorChain = append(validatorChain, survey.Validator(prop.AdditionalValidators[i]))
	}

	if len(validatorChain) > 0 {
		return Validator(survey.ComposeValidators(validatorChain...)), validatorChain
	}

	return NilValidator, validatorChain
}
