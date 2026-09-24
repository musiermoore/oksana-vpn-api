package builder

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/groupby"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/join"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/orderby"
	_select "github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/select"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/update"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/where"
)

func (builder *Builder) queryContext() (*sql.Rows, error) {
	query := builder.buildSelectQuery()

	rows, err := builder.GetDb().QueryContext(
		builder.GetCtx(),
		query,
		builder.GetArgs()...,
	)

	if err != nil {
		builder.printError(query, err)

		return nil, err
	}

	return rows, nil
}

func (r *Builder) execContext(query string, args []any) error {
	_, err := r.db.ExecContext(
		r.ctx,
		query,
		args...,
	)

	if err != nil {
		r.printError(query, err)
	}

	return err
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

	if builder.hasLimit {
		query = append(query, "LIMIT", strconv.Itoa(builder.GetLimit()))
	}

	if builder.hasOffset {
		query = append(query, "OFFSET", strconv.Itoa(builder.GetOffset()))
	}

	return strings.Join(query, " ")
}

func (builder *Builder) buildInsertQuery(columns []string) string {
	placeholders := make([]string, len(columns))

	for i := range placeholders {
		placeholders[i] = "?"
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		builder.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
}

func (builder *Builder) buildUpdateQuery(columns []string) string {
	var query []string

	query = append(query, "UPDATE", builder.GetTable())

	if len(builder.GetJoins()) > 0 {
		query = append(query, join.BuildJoins(builder.GetJoins()))
	}

	query = append(query, "SET", update.BuildSetValues(columns))

	if len(builder.GetWheres()) > 0 {
		query = append(query, "WHERE", where.BuildWheres(builder.GetWheres()))
	}

	if len(builder.GetOrderBy()) > 0 {
		query = append(query, "ORDER BY", orderby.BuildOrderBys(builder.GetOrderBy()))
	}

	if builder.hasLimit {
		query = append(query, "LIMIT", strconv.Itoa(builder.GetLimit()))
	}

	return strings.Join(query, " ")
}

func (builder *Builder) buildDeleteQuery() string {
	var query []string

	query = append(query, "DELETE FROM", builder.GetTable())

	if len(builder.GetWheres()) > 0 {
		query = append(query, "WHERE", where.BuildWheres(builder.GetWheres()))
	}

	if len(builder.GetOrderBy()) > 0 {
		query = append(query, "ORDER BY", orderby.BuildOrderBys(builder.GetOrderBy()))
	}

	if builder.hasLimit {
		query = append(query, "LIMIT", strconv.Itoa(builder.GetLimit()))
	}

	return strings.Join(query, " ")
}

func scanStruct(rows *sql.Rows, dest any) error {
	value := reflect.ValueOf(dest)

	if value.Kind() != reflect.Pointer || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to struct")
	}

	value = value.Elem()
	typeOf := value.Type()

	fields := make(map[string]int)

	for i := 0; i < typeOf.NumField(); i++ {
		field := typeOf.Field(i)
		column := field.Tag.Get("db")

		if column == "" || column == "-" {
			continue
		}

		fields[column] = i
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	scanArgs := make([]any, len(columns))

	for i, column := range columns {
		fieldIndex, ok := fields[column]
		if !ok {
			return fmt.Errorf("no field found for column %q", column)
		}

		scanArgs[i] = value.
			Field(fieldIndex).
			Addr().
			Interface()
	}

	return rows.Scan(scanArgs...)
}
