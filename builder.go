package orm

import "github.com/maxshaw/orm/qb"

type Builder struct {
	executor Executor

	table string

	cols  []string
	exprs []qb.Expr
	args  []any

	offset, limit int

	joins []string
}

func NewBuilder(executor Executor, table string) *Builder {
	return (&Builder{executor: executor, table: table}).reset()
}
