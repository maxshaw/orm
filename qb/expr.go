package qb

import (
	"fmt"
	"strings"
)

type Expr interface {
	Sub() bool
	Build(table string) (string, []any, error)
}

type WhereExpr struct {
	col     string
	args    []any
	op, raw string

	executor func(table string) (string, []any, error)
}

func (w WhereExpr) String() string {
	return w.raw
}

func (w WhereExpr) Sub() bool {
	return false
}

func (w WhereExpr) Build(table string) (cond string, args []any, err error) {
	if w.col != "" {
		col := Quote(table, w.col)
		if w.raw == "" {
			return col + " " + w.op + " ?", w.args, nil
		}
		return col + w.raw, w.args, nil
	}

	if w.executor != nil {
		return w.executor(table)
	}

	if w.raw == "" {
		return "<not a valid expr>", nil, nil
	}
	return w.raw, w.args, nil
}

type subExpr struct {
	typ   string
	exprs []Expr
}

func (subExpr) Sub() bool {
	return true
}

func (e subExpr) Build(table string) (string, []any, error) {
	return Build(table, e.typ, true, e.exprs...)
}

func Quote(a, b string) string {
	if b == "" {
		return fmt.Sprintf("`%s`", a)
	}

	b = strings.TrimPrefix(b, "`")

	if strings.Contains(b, ".") {
		return b
	}

	return fmt.Sprintf("`%s`.`%s`", a, b)
}
