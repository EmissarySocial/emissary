package mockdb

import (
	"reflect"
)

func populateInterface(source interface{}, target interface{}) error {

	sourceValue := reflect.ValueOf(source)

	if sourceValue.Kind() == reflect.Ptr {
		sourceValue = reflect.Indirect(sourceValue)
	}

	sourceType := sourceValue.Type()

	targetValue := reflect.Indirect(reflect.ValueOf(target))

	for index := 0; index < sourceType.NumField(); index = index + 1 {

		sourceField := sourceType.FieldByIndex([]int{index})

		if targetField := targetValue.FieldByName(sourceField.Name); targetField.CanSet() {
			targetField.Set(sourceValue.FieldByName(sourceField.Name))
		}
	}

	return nil
}
