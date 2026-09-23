package select_query

import "strings"

func BuildSelect(selectRows []string) string {
	if len(selectRows) == 0 {
		return "*"
	}

	var rows []string

	for _, row := range selectRows {
		rows = append(rows, row)
	}

	return strings.Join(rows, ", ")
}
