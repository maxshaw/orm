package orm

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/maxshaw/orm/qb"
)

func (b *Builder) UpdateOne(values qb.H) (string, []any, error) {
	return b.Limit(1).Update(values)
}

func (b *Builder) Update(values qb.H) (string, []any, error) {
	cond, whereArgs, err := qb.Build(b.table, "AND", false, b.exprs...)
	if err != nil {
		return "", nil, err
	}

	if cond == "" {
		return "", nil, errors.New("not allow updating rows with no where conditions")
	}

	var sb strings.Builder

	sb.WriteString("UPDATE ")
	sb.WriteString(qb.Quote(b.table, ""))
	sb.WriteString(" SET")

	var i = 0
	for k, v := range values {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(" ")
		sb.WriteString(qb.Quote(b.table, k))
		sb.WriteString(" = ?")

		b.args = append(b.args, v)
		i++
	}

	if cond != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(cond)
		b.args = append(b.args, whereArgs...)
	}

	if b.limit > 0 {
		sb.WriteString(" LIMIT ")
		sb.WriteString(strconv.Itoa(b.limit))
	}

	sq := sb.String()

	log.Printf("[SQL] %s\n", sq)
	log.Printf("[SQL] %+v\n", b.args)

	return sq, b.args, nil
}
