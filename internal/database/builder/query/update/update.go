package update

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/musiermoore/oksana-vpn-api/internal/helpers/strings"
)

func BuildSetValues(columns []string) string {
	return strings.ConcatStrings(columns, ", ", buildSetValue)
}

func buildSetValue(column string) string {
	return fmt.Sprintf("%s = ?", column)
}

func UpdateValues(value any, updatedColumns []string) ([]string, []any, error) {
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

		canBeUpdated := slices.Contains(updatedColumns, column)

		if column == "" || column == "-" || !canBeUpdated {
			continue
		}

		columns = append(columns, column)
		values = append(values, v.Field(i).Interface())
	}

	return columns, values, nil
}
