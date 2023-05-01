package orm

import (
	"log"
	"strings"

	"github.com/maxshaw/orm/qb"
)

func (b *Builder) Insert(value qb.H) (string, []any, error) {
	return b.InsertMulti([]qb.H{value})
}

func (b *Builder) InsertMulti(values []qb.H) (string, []any, error) {
	var sb strings.Builder

	sb.WriteString("INSERT INTO ")
	sb.WriteString(qb.Quote(b.table, ""))
	sb.WriteString(" (")

	var (
		holders string
		columns []string
	)
	for row, value := range values {
		if row == 0 {
			var vb strings.Builder
			vb.WriteString("(")

			var (
				col   int
				count = len(value) - 1
			)
			for k, v := range value {
				columns = append(columns, k)

				sb.WriteString(qb.Quote(k, ""))
				vb.WriteString("?")

				if col == count {
					sb.WriteString(")")
					vb.WriteString(")")
				} else {
					sb.WriteString(", ")
					vb.WriteString(", ")
				}

				b.args = append(b.args, v)
				col++
			}

			holders = vb.String()

			sb.WriteString(" VALUES ")
			sb.WriteString(holders)
		} else {
			sb.WriteString(", ")
			sb.WriteString(holders)

			for _, k := range columns {
				b.args = append(b.args, value[k])
			}
		}
	}

	sq := sb.String()

	log.Printf("[SQL] %s\n", sq)
	log.Printf("[SQL] %+v\n", b.args)

	return sq, b.args, nil
}
