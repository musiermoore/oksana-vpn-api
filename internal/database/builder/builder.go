package builder

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/expression"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/groupby"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/insert"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/join"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/orderby"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/where"
)

func Query(db *sql.DB, ctx context.Context) *Builder {
	return &Builder{
		db:  db,
		ctx: ctx,
	}
}

func (r *Builder) GetDb() *sql.DB {
	return r.db
}

func (r *Builder) GetCtx() context.Context {
	return r.ctx
}

func (r *Builder) GetTable() string {
	return r.table
}

func (r *Builder) GetSelectRows() []string {
	return r.selectRows
}

func (r *Builder) GetJoins() []join.Join {
	return r.joins
}

func (r *Builder) GetWheres() []where.Where {
	return r.wheres
}

func (r *Builder) GetGroupBy() []groupby.GroupBy {
	return r.groupBy
}

func (r *Builder) GetOrderBy() []orderby.OrderBy {
	return r.orderBy
}

func (r *Builder) GetLimit() int {
	if !r.hasLimit {
		return -1
	}

	return r.limit
}

func (r *Builder) GetOffset() int {
	return r.offset
}

func (r *Builder) GetArgs() []any {
	return r.args
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
	r.BaseJoin(
		join.InnerJoin,
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
	r.BaseJoin(
		join.LeftJoin,
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
	r.BaseJoin(
		join.RightJoin,
		table,
		firstColumn,
		operator,
		secondColumn,
	)

	return r
}

func (r *Builder) BaseJoin(
	joinType join.JoinType,
	table string,
	firstColumn string,
	operator string,
	secondColumn string,
) {
	r.joins = append(r.joins, join.Join{
		JoinType:     joinType,
		Table:        table,
		FirstColumn:  firstColumn,
		Operator:     operator,
		SecondColumn: secondColumn,
	})
}

func (r *Builder) Where(column, operator string, value any) *Builder {
	return r.where("AND", column, operator, value)
}

func (r *Builder) OrWhere(column, operator string, value any) *Builder {
	return r.where("OR", column, operator, value)
}

func (r *Builder) where(boolean, column, operator string, value any) *Builder {
	r.wheres = append(r.wheres, where.Where{
		Column:   column,
		Operator: operator,
		Value:    value,
		Boolean:  boolean,
	})

	if _, ok := value.(expression.RawExpression); !ok {
		r.args = append(r.args, value)
	}

	return r
}

func (r *Builder) OrderBy(
	column string,
	ascending bool,
) *Builder {
	r.orderBy = append(r.orderBy, orderby.OrderBy{
		Column:    column,
		Ascending: ascending,
	})

	return r
}

func (r *Builder) Limit(limit int) *Builder {
	r.limit = limit
	r.hasLimit = true

	return r
}

func (r *Builder) Offset(offset int) *Builder {
	r.offset = offset

	return r
}

func (r *Builder) GroupBy(
	column string,
) *Builder {
	r.groupBy = append(r.groupBy, groupby.GroupBy{
		Column: column,
	})

	return r
}

func (r *Builder) First(dest any) error {
	r.Limit(1)

	rows, err := r.QueryContext()
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
	rows, err := r.QueryContext()
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

func (r *Builder) Insert(value any) error {
	columns, values, err := insert.InsertValues(value)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(
		r.ctx,
		r.buildInsertQuery(columns),
		values...,
	)

	return err
}

func (r *Builder) Delete() error {
	_, err := r.db.ExecContext(
		r.ctx,
		r.buildDeleteQuery(),
		r.GetArgs()...,
	)

	return err
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
