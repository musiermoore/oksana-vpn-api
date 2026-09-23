package insert

import (
	"fmt"
	"reflect"
)

func InsertValues(value any) ([]string, []any, error) {
	v := reflect.ValueOf(value)

	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("value must be a struct")
	}

	t := v.Type()

	var columns []string
	var values []any

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		column := field.Tag.Get("db")

		if column == "" || column == "-" {
			continue
		}

		columns = append(columns, column)
		values = append(values, v.Field(i).Interface())
	}

	return columns, values, nil
}
