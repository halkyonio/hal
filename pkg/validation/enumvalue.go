package validation

import (
	"fmt"
	"strings"
)

type EnumValue struct {
	Value       string
	values      map[string]bool
	valueName   string
	knownValues string
}

func (e EnumValue) String() string {
	return e.Value
}
func NewEnumValue(valueName string, values ...fmt.Stringer) EnumValue {
	if len(values) > 0 {
		valueMap := make(map[string]bool, len(values))
		for _, value := range values {
			valueMap[value.String()] = true
		}
		set := EnumValue{
			values:    valueMap,
			valueName: valueName,
		}
		set.GetKnownValues() // initialize known values
		return set
	}
	panic(fmt.Errorf("a EnumValue must contain at least one possible value"))
}
func (e EnumValue) Contains(ans interface{}) error {
	var value string
	switch v := ans.(type) {
	case fmt.Stringer:
		value = v.String()
	case string:
		value = v
	default:
		return fmt.Errorf("can only validate string or Stringer instances, was given: %v", ans)
	}

	if !e.values[value] {
		return fmt.Errorf("unknown %s: %s, valid %ss are: %s", e.valueName, value, e.valueName, e.knownValues)
	}
	return nil
}
func (e *EnumValue) GetKnownValues() string {
	if len(e.knownValues) == 0 {
		values := make([]string, 0, len(e.values))
		for value := range e.values {
			values = append(values, value)
		}
		e.knownValues = strings.Join(values, ",")
	}
	return e.knownValues
}
