package orm

import (
	"log"
	"strconv"
	"strings"

	"github.com/maxshaw/orm/qb"
)

func (b *Builder) JoinUsing(target, col string) *Builder {
	return b.Join(target, col, "")
}

func (b *Builder) Join(target, first, second string) *Builder {
	return b.join("INNER", target, first, second)
}

func (b *Builder) LeftJoinUsing(target, col string) *Builder {
	return b.LeftJoin(target, col, "")
}

func (b *Builder) LeftJoin(target, first, second string) *Builder {
	return b.join("LEFT", target, first, second)
}

func (b *Builder) RightJoinUsing(target, col string) *Builder {
	return b.RightJoin(target, col, "")
}

func (b *Builder) RightJoin(target, first, second string) *Builder {
	return b.join("RIGHT", target, first, second)
}

func (b *Builder) OuterJoinUsing(target, col string) *Builder {
	return b.RightJoin(target, col, "")
}

func (b *Builder) OuterJoin(target, first, second string) *Builder {
	return b.join("OUTER", target, first, second)
}

func (b *Builder) join(typ, target, first, second string) *Builder {
	var sb strings.Builder

	sb.WriteString(" ")
	sb.WriteString(typ)
	sb.WriteString(" JOIN ")
	sb.WriteString(qb.Quote(target, ""))

	if second == "" {
		sb.WriteString(" USING (")
		sb.WriteString(qb.Quote(first, ""))
		sb.WriteString(")")
	} else {
		sb.WriteString(" ON (")
		sb.WriteString(qb.Quote(b.table, first))
		sb.WriteString(" = ")
		sb.WriteString(qb.Quote(target, second))
		sb.WriteString(")")
	}

	b.joins = append(b.joins, sb.String())
	return b
}

func (b *Builder) reset() *Builder {
	b.cols = []string{}
	b.exprs = []qb.Expr{}
	b.args = []any{}

	b.offset = -1
	b.limit = -1

	b.joins = []string{}

	return b
}

func (b *Builder) Select(cols ...string) *Builder {
	if len(cols) > 0 {
		b.cols = cols
	}
	return b
}

func (b *Builder) Where(a ...qb.Expr) *Builder {
	b.exprs = append(b.exprs, a...)
	return b
}

func (b *Builder) Offset(a int) *Builder {
	b.offset = a
	return b
}

func (b *Builder) Limit(a int) *Builder {
	b.limit = a
	return b
}

func (b *Builder) ToSQL() (string, []any, error) {
	cond, whereArgs, err := qb.Build(b.table, "AND", false, b.exprs...)
	if err != nil {
		return "", nil, err
	}

	var sb strings.Builder

	sb.WriteString("SELECT ")

	if len(b.cols) < 1 {
		sb.WriteString("*")
	} else {
		sb.WriteString("`")
		sb.WriteString(b.table)
		sb.WriteString("`.`")
		sb.WriteString(strings.Join(b.cols, "`, `"+b.table+"`.`"))
		sb.WriteString("`")
	}

	sb.WriteString(" FROM `")
	sb.WriteString(b.table)
	sb.WriteString("`")

	for _, join := range b.joins {
		sb.WriteString(join)
		sb.WriteString(" ")
	}

	if cond != "" {
		cond = strings.TrimPrefix(strings.TrimPrefix(cond, " AND "), " OR ")
		if cond != "" {
			sb.WriteString(" WHERE ")
			sb.WriteString(cond)
		}
	}

	if b.limit > -1 {
		sb.WriteString(" LIMIT ")

		if b.offset > -1 {
			sb.WriteString(strconv.Itoa(b.offset))
			sb.WriteString(", ")
		}

		sb.WriteString(strconv.Itoa(b.limit))
	}

	sq, args := sb.String(), append(b.args, whereArgs...)

	log.Printf("[SQL] %s\n", sq)
	log.Printf("[SQL] %+v\n", args)

	b.reset()

	return sq, args, nil
}
