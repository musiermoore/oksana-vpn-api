package orderby

import (
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/bindings"
	"github.com/musiermoore/oksana-vpn-api/internal/helpers/strings"
)

type OrderBy struct {
	Column    string
	Ascending bool
}

func BuildOrderBys(orderBys []OrderBy) string {
	return strings.ConcatStrings(orderBys, ", ", buildOrderBy)
}

func buildOrderBy(orderBy OrderBy) string {
	return bindings.Bindings("? ?",
		orderBy.Column,
		GetAscending(orderBy.Ascending),
	)
}

func GetAscending(asc bool) string {
	ascending := "ASC"

	if !asc {
		ascending = "DESC"
	}

	return ascending
}
