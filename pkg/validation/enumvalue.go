package validation

import (
	"fmt"
	"reflect"
	"strings"
)

type EnumValue struct {
	Provided    string
	Value       interface{}
	values      map[string]interface{}
	valueName   string
	knownValues string
	typeDef     reflect.Type
}

func (e EnumValue) String() string {
	if e.Value != nil {
		return e.Value.(fmt.Stringer).String()
	}
	return e.Provided
}

func NewEnumValue(valueName string, values ...interface{}) EnumValue {
	if len(values) > 0 {
		valueMap := make(map[string]interface{}, len(values))
		var seen reflect.Type = nil
		for _, value := range values {
			t := reflect.TypeOf(value)
			if seen == nil {
				seen = t
			} else if seen != t {
				panic(fmt.Errorf("an EnumValue can only contain one type of values"))
			}
			s, err := valueAsString(value)
			if err != nil {
				panic(err)
			}
			valueMap[s] = value
		}
		set := EnumValue{
			values:    valueMap,
			valueName: valueName,
			typeDef:   seen,
		}
		set.GetKnownValues() // initialize known values
		return set
	}
	panic(fmt.Errorf("an EnumValue must contain at least one possible value"))
}

func (e EnumValue) IsProvidedValid() bool {
	return len(e.Provided) > 0 && e.values[e.Provided] != nil
}

func (e EnumValue) Contains(ans interface{}) error {
	value, err := valueAsString(ans)
	if err != nil {
		return err
	}
	_, ok := e.values[value]
	if !ok {
		return fmt.Errorf("unknown %s: '%s', valid %ss are: %s", e.valueName, value, e.valueName, e.knownValues)
	}
	return nil
}

func valueAsString(ans interface{}) (string, error) {
	switch v := ans.(type) {
	case fmt.Stringer:
		return v.String(), nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("can only validate string or Stringer instances, was given: %v", ans)
	}
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

func (e *EnumValue) Set(value interface{}) error {
	if err := e.acceptValue(value); err != nil {
		return err
	}
	e.Value = value
	return nil
}

func (e *EnumValue) MustSet(value interface{}) {
	if err := e.Set(value); err != nil {
		panic(err)
	}
}

func (e *EnumValue) Get() interface{} {
	if e.Value != nil {
		return e.Value
	}
	return e.values[e.Provided]
}

func (e *EnumValue) acceptValue(value interface{}) error {
	valueType := reflect.TypeOf(value)
	if valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
	}
	if e.typeDef != valueType {
		return fmt.Errorf("impossible to convert provided %s to %s", valueType.Name(), e.typeDef.Name())
	}

	return e.Contains(value)
}
