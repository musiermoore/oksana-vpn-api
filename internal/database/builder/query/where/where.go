package where

import (
	"fmt"
	"strings"

	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/expression"
)

type Where struct {
	Column   string
	Operator string
	Value    any
	Boolean  string
}

func BuildWheres(wheres []Where) string {
	var parts []string

	for i, where := range wheres {
		condition := buildWhere(where)

		if i > 0 {
			condition = where.Boolean + " " + condition
		}

		parts = append(parts, condition)
	}

	return strings.Join(parts, " ")
}

func buildWhere(where Where) string {
	if raw, ok := where.Value.(expression.RawExpression); ok {
		return fmt.Sprintf(
			"%s %s %s",
			where.Column,
			where.Operator,
			raw.Column,
		)
	}

	return fmt.Sprintf(
		"%s %s ?",
		where.Column,
		where.Operator,
	)
}
