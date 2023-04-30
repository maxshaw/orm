package qb

import (
	"reflect"
	"strings"
)

func Eq(col string, val any) Expr {
	return WhereExpr{col: col, op: "=", args: []any{val}}
}

func Neq(col string, val any) Expr {
	return WhereExpr{col: col, op: "<>", args: []any{val}}
}

func Gt(col string, val any) Expr {
	return WhereExpr{col: col, op: ">", args: []any{val}}
}

func Lt(col string, val any) Expr {
	return WhereExpr{col: col, op: "<", args: []any{val}}
}

func Gte(col string, val any) Expr {
	return WhereExpr{col: col, op: ">=", args: []any{val}}
}

func Lte(col string, val any) Expr {
	return WhereExpr{col: col, op: "<=", args: []any{val}}
}

func Between(col string, a, b any) Expr {
	return WhereExpr{col: col, raw: " BETWEEN ? AND ?", args: []any{a, b}}
}

func Raw(s string, args ...any) Expr {
	return WhereExpr{raw: s, args: args}
}

func Null(col string) Expr {
	return WhereExpr{col: col, raw: " IS NULL"}
}

func NotNull(col string) Expr {
	return WhereExpr{col: col, raw: " IS NOT NULL"}
}

func In(col string, args ...any) Expr {
	return WhereExpr{executor: func(table string) (string, []any, error) {
		var raw strings.Builder
		for _, arg := range args {
			if reflect.TypeOf(arg).Kind() == reflect.Slice {
				rv := reflect.ValueOf(arg)
				args = make([]any, 0)
				for i := 0; i < rv.Len(); i++ {
					raw.WriteString(", ?")
					args = append(args, rv.Index(i).Interface())
				}
				break
			}
		}

		if raw.Len() < 1 {
			for range args {
				raw.WriteString(", ?")
			}
		}

		return Quote(table, col) + " IN (" + raw.String()[2:] + ")", args, nil
	}}
}

func Like(col string, val string) Expr {
	return WhereExpr{col: col, op: "LIKE", args: []any{"%" + val + "%"}}
}

func RLike(col string, val string) Expr {
	return WhereExpr{col: col, op: "LIKE", args: []any{val + "%"}}
}

func LLike(col string, val string) Expr {
	return WhereExpr{col: col, op: "LIKE", args: []any{"%" + val}}
}

func And(a ...Expr) Expr {
	return subExpr{typ: "AND", exprs: a}
}

func Or(a ...Expr) Expr {
	return subExpr{typ: "OR", exprs: a}
}

func Build(table, typ string, sub bool, a ...Expr) (string, []any, error) {
	var (
		sb      strings.Builder
		args    []any
		typStr      = " " + typ + " "
		opened  int = -1
		bracket bool
	)

	if sub {
		for _, e := range a {
			if e.Sub() {
				bracket = true
				break
			}
		}
	}

	for i, e := range a {
		if e.Sub() {
			if bracket && opened > -1 {
				sb.WriteString(")")
				opened = -1
			}

			if i > 0 {
				sb.WriteString(typStr)
			}
		} else {
			if opened == -1 {
				if i > 0 {
					sb.WriteString(typStr)
				}

				if bracket {
					sb.WriteString("(")
				}
			} else if opened > -1 {
				sb.WriteString(typStr)
			}

			if bracket {
				opened++
			}
		}

		out, whereArgs, err := e.Build(table)
		if err != nil {
			return "", nil, err
		}

		args = append(args, whereArgs...)
		sb.WriteString(out)
	}

	if opened > 0 {
		sb.WriteString(")")
	}

	sq := sb.String()

	if sub && len(a) > 1 {
		sq = "(" + sq + ")"
	}

	return sq, args, nil
}
