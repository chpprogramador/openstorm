package dialects

import (
	"fmt"
	"strings"
)

type DialectType string

const (
	Postgres  DialectType = "postgres"
	SQLServer DialectType = "sqlserver"
	MySQL     DialectType = "mysql"
	Access    DialectType = "access"
)

// SQLDialect interface define o contrato para os dialetos suportados
// type SQLDialect interface {
// 	FetchBatches(db *sql.DB, job models.Job) ([][][]any, error)
// 	InsertBatch(db *sql.DB, insertSQL string, batch [][]any) error
// }

// NewDialect retorna a implementação apropriada do SQLDialect baseado no tipo
func NewDialect(dbType string) (SQLDialect, error) {
	switch DialectType(strings.ToLower(dbType)) {
	case Postgres:
		return PostgresDialect{}, nil
	case SQLServer:
		return SQLServerDialect{}, nil
	case MySQL:
		return MySQLDialect{}, nil
	case Access:
		return AccessDialect{}, nil
	default:
		return nil, fmt.Errorf("dialeto desconhecido: %s", dbType)
	}
}
