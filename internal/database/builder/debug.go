package builder

import (
	"fmt"

	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/bindings"
)

func (r *Builder) PrintSQL() *Builder {
	r.printSql(false)

	return r
}

func (r *Builder) PrintRawSQL() *Builder {
	r.printSql(true)

	return r
}

func (builder *Builder) GetSql(filled bool) string {
	sql := builder.buildSelectQuery()

	if filled {
		sql = bindings.Bindings(sql, builder.GetArgs()...)
	}

	return sql
}

func (builder *Builder) printSql(filled bool) {
	fmt.Println("SQL debug: " + builder.GetSql(filled))
}

func (r *Builder) printError(query string, err error) {
	fmt.Printf("SQL error: %v\nQuery: %s\nBindings: %s\n", err, query, r.GetArgs())
}
