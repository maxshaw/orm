package orm

import "github.com/maxshaw/orm/qb"

type Builder struct {
	executor Executor

	table string

	args  []any
	exprs []qb.Expr

	cols []string

	offset, limit int

	order, group string

	having []qb.Expr

	joins []string
}

func NewBuilder(executor Executor, table string) *Builder {
	return (&Builder{executor: executor, table: table}).reset()
}
