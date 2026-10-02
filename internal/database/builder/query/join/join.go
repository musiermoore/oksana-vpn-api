package join

import (
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/bindings"
	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/expression"
	"github.com/musiermoore/oksana-vpn-api/internal/helpers/strings"
)

type JoinType string
type ConditionType string

const (
	InnerJoin JoinType = "INNER"
	FullJoin  JoinType = "FULL"
	CrossJoin JoinType = "CROSS"
	LeftJoin  JoinType = "LEFT"
	RightJoin JoinType = "RIGHT"
)

const (
	ConditionTypeColumn ConditionType = "column"
	ConditionTypeValue  ConditionType = "value"
	ConditionTypeRaw    ConditionType = "raw"
	ConditionTypeGroup  ConditionType = "group"
)

type Builder struct {
	conditions []Condition
}

type Condition struct {
	Boolean string
	Type    ConditionType

	FirstColumn  string
	Operator     string
	SecondColumn string

	Column string
	Value  any

	Raw   any
	Group *Builder
}

type Join struct {
	JoinType     JoinType
	Table        string
	FirstColumn  string
	Operator     string
	SecondColumn string

	Group *Builder
}

func NewBuilder() *Builder {
	return &Builder{
		conditions: []Condition{},
	}
}

func (b *Builder) On(firstColumn string, operator string, secondColumn string) *Builder {
	b.conditions = append(b.conditions, Condition{
		Boolean:      "AND",
		Type:         ConditionTypeColumn,
		FirstColumn:  firstColumn,
		Operator:     operator,
		SecondColumn: secondColumn,
	})

	return b
}

func (b *Builder) OrOn(firstColumn string, operator string, secondColumn string) *Builder {
	b.conditions = append(b.conditions, Condition{
		Boolean:      "OR",
		Type:         ConditionTypeColumn,
		FirstColumn:  firstColumn,
		Operator:     operator,
		SecondColumn: secondColumn,
	})

	return b
}

func (b *Builder) OnRaw(raw expression.RawExpression) *Builder {
	b.conditions = append(b.conditions, Condition{
		Boolean: "AND",
		Type:    ConditionTypeRaw,
		Raw:     raw,
	})

	return b
}

func (b *Builder) OnGroup(callback func(*Builder)) *Builder {
	return b.addGroup("AND", callback)
}

func (b *Builder) OrOnGroup(callback func(*Builder)) *Builder {
	return b.addGroup("OR", callback)
}

func (b *Builder) addGroup(boolean string, callback func(*Builder)) *Builder {
	group := NewBuilder()
	callback(group)

	b.conditions = append(b.conditions, Condition{
		Boolean: boolean,
		Type:    ConditionTypeGroup,
		Group:   group,
	})

	return b
}

func (b *Builder) Where(column string, operator string, value any) *Builder {
	b.conditions = append(b.conditions, Condition{
		Boolean:  "AND",
		Type:     ConditionTypeRaw,
		Column:   column,
		Operator: operator,
		Value:    value,
	})

	return b
}

func (b *Builder) OrWhere(column string, operator string, value any) *Builder {
	b.conditions = append(b.conditions, Condition{
		Boolean:  "OR",
		Type:     ConditionTypeRaw,
		Column:   column,
		Operator: operator,
		Value:    value,
	})

	return b
}

func (b *Builder) Conditions() []Condition {
	return b.conditions
}

func BuildJoins(joins []Join) string {
	return strings.ConcatStrings(joins, " ", buildJoin)
}

func buildJoin(j Join) string {
	if j.Group != nil {
		return bindings.FormatArgs(
			"%s JOIN %s ON %s",
			j.JoinType,
			j.Table,
			buildConditions(j.Group.Conditions()),
		)
	}

	return bindings.FormatArgs(
		"%s JOIN %s ON %s %s %s",
		j.JoinType,
		j.Table,
		j.FirstColumn,
		j.Operator,
		j.SecondColumn,
	)
}

func buildConditions(conditions []Condition) string {
	parts := make([]string, 0, len(conditions))

	for i, condition := range conditions {
		prefix := ""
		if i > 0 {
			prefix = condition.Boolean + " "
		}

		switch condition.Type {
		case ConditionTypeColumn:
			parts = append(parts, bindings.FormatArgs(
				"%s%s %s %s",
				prefix,
				condition.FirstColumn,
				condition.Operator,
				condition.SecondColumn,
			))

		case ConditionTypeValue:
			parts = append(parts, bindings.FormatArgs(
				"%s%s %s %s",
				prefix,
				condition.Column,
				condition.Operator,
				condition.Value,
			))

		case ConditionTypeRaw:
			parts = append(parts, bindings.FormatArgs(
				"%s%s",
				prefix,
				condition.Raw,
			))

		case ConditionTypeGroup:
			parts = append(parts, bindings.FormatArgs(
				"%s(%s)",
				prefix,
				buildConditions(condition.Group.Conditions()),
			))
		}
	}

	return strings.ConcatStrings(parts, " ", func(part string) string {
		return part
	})
}
