package builder

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/groupby"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/join"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/orderby"
	_select "github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/select"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/where"
)

func (builder *Builder) QueryContext() (*sql.Rows, error) {
	rows, err := builder.GetDb().QueryContext(
		builder.GetCtx(),
		builder.buildSelectQuery(),
		builder.GetArgs()...,
	)

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (builder *Builder) buildSelectQuery() string {
	var query []string

	query = append(query, "SELECT", _select.BuildSelect(builder.GetSelectRows()))
	query = append(query, "FROM", builder.GetTable())

	if len(builder.GetJoins()) > 0 {
		query = append(query, join.BuildJoins(builder.GetJoins()))
	}

	if len(builder.GetWheres()) > 0 {
		query = append(query, "WHERE", where.BuildWheres(builder.GetWheres()))
	}

	if len(builder.GetGroupBy()) > 0 {
		query = append(query, "GROUP BY", groupby.BuildGroupBys(builder.GetGroupBy()))
	}

	if len(builder.GetOrderBy()) > 0 {
		query = append(query, "ORDER BY", orderby.BuildOrderBys(builder.GetOrderBy()))
	}

	if builder.GetLimit() > 0 {
		query = append(query, "LIMIT", strconv.Itoa(builder.GetLimit()))
	}

	if builder.GetOffset() > 0 {
		query = append(query, "OFFSET", strconv.Itoa(builder.GetOffset()))
	}

	return strings.Join(query, " ")
}

func (r *Builder) buildInsertQuery(columns []string) string {
	placeholders := make([]string, len(columns))

	for i := range placeholders {
		placeholders[i] = "?"
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		r.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
}
