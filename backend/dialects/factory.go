package dialects

import (
	"database/sql"
	"etl/models"
	"fmt"
)

type DialectType string

const (
	Postgres  DialectType = "postgres"
	SQLServer DialectType = "sqlserver"
	MySQL     DialectType = "mysql"
	Access    DialectType = "access"
)

func NewDialect(dbType string) (SQLDialect, error) {
	switch DialectType(dbType) {
	case Postgres:
		return PostgresDialect{}, nil
	case SQLServer:
		return SQLServerDialect{}, nil
	case MySQL:
		return MysqlDialect{}, nil
	case Access:
		return AccessDialect{}, nil
	default:
		return nil, fmt.Errorf("dialeto desconhecido: %s", dbType)
	}
}

// SQLDialect interface

type SQLDialect interface {
	FetchBatches(db *sql.DB, job models.Job) ([][][]any, error)
	InsertBatch(db *sql.DB, insertSQL string, batch [][]any) error
}
