package strings

import "strings"

func ConcatStrings[T any](items []T, separator string, callback func(T) string) string {
	var builder strings.Builder

	for i, item := range items {
		if i > 0 {
			builder.WriteString(separator)
		}

		builder.WriteString(callback(item))
	}

	return builder.String()
}
