package step

import (
	"reflect"
)

// Nested returns every list of Steps that a Step holds in its exported fields: each []Step field, and
// each entry of a map[string][]Step field, such as a switch's cases.  Empty lists are left out.
func Nested(current Step) [][]Step {

	result := make([][]Step, 0)
	value := reflect.Indirect(reflect.ValueOf(current))

	if value.Kind() != reflect.Struct {
		return result
	}

	for index := range value.NumField() {

		// Interface() panics on an unexported field, and a step keeps its sub-steps in exported ones
		if !value.Type().Field(index).IsExported() {
			continue
		}

		result = append(result, stepsIn(value.Field(index).Interface())...)
	}

	return result
}

// stepsIn returns the lists of Steps that one field holds, and none for any other kind of field
func stepsIn(field any) [][]Step {

	switch typed := field.(type) {

	case []Step:
		if len(typed) > 0 {
			return [][]Step{typed}
		}

	case map[string][]Step:
		result := make([][]Step, 0, len(typed))

		for _, nested := range typed {
			if len(nested) > 0 {
				result = append(result, nested)
			}
		}

		return result
	}

	return nil
}

// IncludedActions returns the name of every action that a list of Steps runs through include,
// at any depth of nesting, in the order found
func IncludedActions(steps []Step) []string {

	result := make([]string, 0)

	for _, current := range steps {

		if include, ok := current.(Include); ok {
			result = append(result, include.Action)
		}

		for _, nested := range Nested(current) {
			result = append(result, IncludedActions(nested)...)
		}
	}

	return result
}
