package groupby

import "github.com/musiermoore/oksana-vpn-api/internal/helpers/strings"

type GroupBy struct {
	Column string
}

func BuildGroupBys(groupBy []GroupBy) string {
	return strings.ConcatStrings(groupBy, ", ", buildGroupBy)
}

func buildGroupBy(groupBy GroupBy) string {
	return groupBy.Column
}
