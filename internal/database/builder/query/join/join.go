package join

import (
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/bindings"
	"github.com/musiermoore/oksana-vpn-api/internal/helpers/strings"
)

type JoinType string

const (
	InnerJoin JoinType = "INNER"
	FullJoin  JoinType = "FULL"
	CrossJoin JoinType = "CROSS"
	LeftJoin  JoinType = "LEFT"
	RightJoin JoinType = "RIGHT"
)

type Join struct {
	JoinType     JoinType
	Table        string
	FirstColumn  string
	Operator     string
	SecondColumn string
}

func BuildJoins(joins []Join) string {
	return strings.ConcatStrings(joins, " ", buildJoin)
}

func buildJoin(join Join) string {
	return bindings.FormatArgs(
		"%s JOIN %s ON %s %s %s",
		join.JoinType,
		join.Table,
		join.FirstColumn,
		join.Operator,
		join.SecondColumn,
	)
}
