package dialects

import (
	"fmt"
	"strings"
)

type DialectType string

const (
	Postgres DialectType = "postgres"
)

// NewDialect retorna a implementação apropriada do SQLDialect baseado no tipo
func NewDialect(dbType string) (SQLDialect, error) {
	switch DialectType(strings.ToLower(dbType)) {
	case Postgres:
		return PostgresDialect{}, nil
	default:
		return nil, fmt.Errorf("dialeto desconhecido: %s", dbType)
	}
}
