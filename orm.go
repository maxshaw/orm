package orm

import "database/sql"

type Executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
}

type Modeler interface {
	TableName() string
}
