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
	db  *sql.DB
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

	args []any
}
