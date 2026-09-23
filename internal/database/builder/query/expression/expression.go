package expression

type RawExpression struct {
	Column string
	Args   []any
}

func Raw(column string, args ...any) RawExpression {
	return RawExpression{
		Column: column,
		Args:   args,
	}
}
