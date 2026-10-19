package builder

import (
    "context"
    "database/sql"

    "github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/groupby"
    "github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/join"
    "github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/orderby"
    "github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/where"
)

type Builder struct {
    db  QueryExecutor
    ctx context.Context

    table      string
    selectRows []string
    joins      []join.Join
    wheres     []where.Where

    groupBy []groupby.GroupBy
    orderBy []orderby.OrderBy

    hasLimit bool
    limit    int

    hasOffset bool
    offset    int
    forUpdate bool

    args []any
}

type QueryExecutor interface {
    QueryContext(context.Context, string, ...any) (*sql.Rows, error)
    ExecContext(context.Context, string, ...any) (sql.Result, error)
}
