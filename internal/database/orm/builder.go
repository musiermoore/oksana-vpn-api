package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Where struct {
	column   string
	operator string
	value    any
	boolean  string
}

type JoinType string

const (
	InnerJoin JoinType = "INNER"
	FullJoin  JoinType = "FULL"
	CrossJoin JoinType = "CROSS"
	LeftJoin  JoinType = "LEFT"
	RightJoin JoinType = "RIGHT"
)

type Join struct {
	joinType     JoinType
	table        string
	firstColumn  string
	operator     string
	secondColumn string
}

type OrderBy struct {
	column    string
	ascending bool
}

type GroupBy struct {
	column string
}

type Builder struct {
	db  *sql.DB
	ctx context.Context

	table      string
	selectRows []string
	joins      []Join
	wheres     []Where

	groupBy []GroupBy
	orderBy []OrderBy
	limit   int
	offset  int

	args []any
}

func Query(db *sql.DB, ctx context.Context) *Builder {
	return &Builder{
		db:  db,
		ctx: ctx,
	}
}

func (r *Builder) Table(table string) *Builder {
	r.table = table

	return r
}

func (r *Builder) Select(columns ...string) *Builder {
	r.selectRows = append(r.selectRows, columns...)

	return r
}

func (r *Builder) InnerJoin(
	table string,
	firstColumn string,
	operator string,
	secondColumn string,
) *Builder {
	r.join(
		InnerJoin,
		table,
		firstColumn,
		operator,
		secondColumn,
	)

	return r
}

func (r *Builder) LeftJoin(
	table string,
	firstColumn string,
	operator string,
	secondColumn string,
) *Builder {
	r.join(
		LeftJoin,
		table,
		firstColumn,
		operator,
		secondColumn,
	)

	return r
}

func (r *Builder) RightJoin(
	table string,
	firstColumn string,
	operator string,
	secondColumn string,
) *Builder {
	r.join(
		RightJoin,
		table,
		firstColumn,
		operator,
		secondColumn,
	)

	return r
}

func (r *Builder) join(
	joinType JoinType,
	table string,
	firstColumn string,
	operator string,
	secondColumn string,
) {
	r.joins = append(r.joins, Join{
		joinType:     joinType,
		table:        table,
		firstColumn:  r.prepareBinding(firstColumn),
		operator:     operator,
		secondColumn: r.prepareBinding(secondColumn),
	})
}

func (r *Builder) Where(column, operator string, value any) *Builder {
	return r.where("AND", column, operator, value)
}

func (r *Builder) OrWhere(column, operator string, value any) *Builder {
	return r.where("OR", column, operator, value)
}

func (r *Builder) where(boolean, column, operator string, value any) *Builder {
	r.wheres = append(r.wheres, Where{
		column:   column,
		operator: operator,
		value:    value,
		boolean:  boolean,
	})

	r.args = append(r.args, value)

	return r
}

func (r *Builder) OrderBy(
	column string,
	ascending bool,
) *Builder {
	r.orderBy = append(r.orderBy, OrderBy{
		column:    column,
		ascending: ascending,
	})

	return r
}

func (r *Builder) Limit(limit int) *Builder {
	r.limit = limit

	return r
}

func (r *Builder) Offset(offset int) *Builder {
	r.offset = offset

	return r
}

func (r *Builder) GroupBy(
	column string,
) *Builder {
	r.groupBy = append(r.groupBy, GroupBy{
		column: column,
	})

	return r
}

func (r *Builder) First(dest any) error {
	r.Limit(1)

	rows, err := r.queryContext()
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return sql.ErrNoRows
	}

	return scanStruct(rows, dest)
}

func (r *Builder) List(dest any) error {
	rows, err := r.queryContext()
	if err != nil {
		return err
	}
	defer rows.Close()

	value := reflect.ValueOf(dest)

	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("destination must be a pointer to slice")
	}

	slice := value.Elem()
	elementType := slice.Type().Elem()

	for rows.Next() {
		element := reflect.New(elementType)

		if err := scanStruct(rows, element.Interface()); err != nil {
			return err
		}

		slice.Set(reflect.Append(slice, element.Elem()))
	}

	return rows.Err()
}

func (r *Builder) queryContext() (*sql.Rows, error) {
	rows, err := r.db.QueryContext(
		r.ctx,
		r.buildSelectQuery(),
		r.args...,
	)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *Builder) buildSelectQuery() string {
	var query []string

	query = append(query, "SELECT")
	query = append(query, r.buildSelect())
	query = append(query, "FROM")
	query = append(query, r.table)

	if len(r.joins) > 0 {
		query = append(query, r.buildJoins())
	}

	if len(r.wheres) > 0 {
		query = append(query, "WHERE", r.buildWheres())
	}

	if len(r.groupBy) > 0 {
		query = append(query, "GROUP BY", r.buildGroupBys())
	}

	if len(r.orderBy) > 0 {
		query = append(query, "ORDER BY", r.buildOrderBys())
	}

	if r.limit > 0 {
		query = append(query, "LIMIT", strconv.Itoa(r.limit))
	}

	if r.offset > 0 {
		query = append(query, "OFFSET", strconv.Itoa(r.offset))
	}

	return strings.Join(query, " ")
}

func (r *Builder) buildSelect() string {
	var rows []string

	for _, row := range r.selectRows {
		rows = append(rows, fmt.Sprintf(`%s`, row))
	}

	return strings.Join(rows, ", ")
}

func (r *Builder) buildJoins() string {
	return concatStrings(r.joins, " ", r.buildJoin)
}

func (r *Builder) buildJoin(join Join) string {
	return r.bindings(
		`%s JOIN %s ON %s %s %s`,
		join.joinType,
		join.table,
		join.firstColumn,
		join.operator,
		join.secondColumn,
	)
}

func (r *Builder) buildWheres() string {
	var parts []string

	for i, where := range r.wheres {
		condition := r.buildWhere(where)

		if i > 0 {
			condition = where.boolean + " " + condition
		}

		parts = append(parts, condition)
	}

	return strings.Join(parts, " ")
}

func (r *Builder) buildWhere(where Where) string {
	return r.formatArgs(
		"%s %s ?",
		where.column,
		where.operator,
	)
}

func (r *Builder) buildOrderBys() string {
	return concatStrings(r.orderBy, ", ", r.buildOrderBy)
}

func (r *Builder) buildOrderBy(orderBy OrderBy) string {
	return r.bindings(`? ?`,
		orderBy.column,
		getAscending(orderBy.ascending),
	)
}

func getAscending(asc bool) string {
	ascending := "ASC"

	if !asc {
		ascending = "DESC"
	}

	return ascending
}

func (r *Builder) buildGroupBys() string {
	return concatStrings(r.groupBy, ", ", r.buildGroupBy)
}

func (r *Builder) buildGroupBy(groupBy GroupBy) string {
	return groupBy.column
}

func (r *Builder) formatArgs(str string, args ...any) string {
	return fmt.Sprintf(str, args...)
}

func (r *Builder) bindings(str string, args ...any) string {
	fmt.Println(strings.ReplaceAll(str, "?", "%s"))
	return fmt.Sprintf(strings.ReplaceAll(str, "?", "%s"), r.prepareBindings(args)...)
}

func (r *Builder) prepareBindings(args ...any) []any {
	var bindings []any

	for _, item := range args {
		bindings = append(bindings, r.prepareBinding(item))
	}

	return bindings
}

func (r *Builder) prepareBinding(arg any) string {
	switch v := arg.(type) {
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)

	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)

	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)

	case string:
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"

	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"

	case nil:
		return "NULL"

	default:
		return fmt.Sprintf("'%v'", v)
	}
}

func concatStrings[T any](items []T, separator string, callback func(T) string) string {
	var builder strings.Builder

	for i, item := range items {
		if i > 0 {
			builder.WriteString(separator)
		}

		builder.WriteString(callback(item))
	}

	return builder.String()
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
