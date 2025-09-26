package dialects

import (
	"database/sql"
	"etl/models"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type SQLDialect interface {
	FetchTotalCount(db *sql.DB, job models.Job) (int, error)
	BuildInsertQuery(job models.Job, records []map[string]interface{}) (string, []interface{})
	BuildSelectQueryByHash(job models.Job, concurrencyIndex, totalConcurrency int, mainTable string) string
	BuildExplainSelectQueryByHash(job models.Job) string
}

type PostgresDialect struct{}

func (d PostgresDialect) FetchTotalCount(db *sql.DB, job models.Job) (int, error) {
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_subquery", job.SelectSQL)

	var count int
	err := db.QueryRow(countSQL).Scan(&count)
	return count, err
}

func (d PostgresDialect) BuildSelectQueryByHash(job models.Job, concurrencyIndex, totalConcurrency int, mainTable string) string {
	withWhere, modifiedSQL := AnalyzeAndModifySQL(job.SelectSQL)

	// Usa hashtextextended (mais robusto que hashtext simples)
	hashExpr := fmt.Sprintf("abs(mod(hashtextextended(%s, 0), %d)) = %d",
		mainTable+".ctid::text", totalConcurrency, concurrencyIndex)

	var queryRet string
	if withWhere {
		queryRet = fmt.Sprintf("%s AND (%s)", modifiedSQL, hashExpr)
	} else {
		queryRet = fmt.Sprintf("%s WHERE (%s)", modifiedSQL, hashExpr)
	}

	println("Query com hash:", queryRet)

	return queryRet
}

func (d PostgresDialect) BuildExplainSelectQueryByHash(job models.Job) string {
	return "EXPLAIN (FORMAT JSON, VERBOSE) " + job.SelectSQL
}

// AnalyzeAndModifySQL detecta se a query tem WHERE e remove LIMIT e ORDER BY (considerando subqueries).
func AnalyzeAndModifySQL(query string) (bool, string) {
	lowerQuery := strings.ToLower(query)

	// Detecta se há WHERE
	hasWhere := regexp.MustCompile(`\bwhere\b`).MatchString(lowerQuery)

	// Remove LIMIT
	reLimit := regexp.MustCompile(`(?i)\blimit\s+\d+(\s*,\s*\d+)?\b`)
	queryNoLimit := reLimit.ReplaceAllString(query, "")

	// Remove ORDER BY somente no nível principal
	queryNoOrder := removeOrderByAtTopLevel(queryNoLimit)

	// Limpa espaços extras
	queryModified := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(queryNoOrder, " "))

	fmt.Printf("Query modificada: %d caracteres: %s\n", len([]rune(queryModified)), queryModified)
	fmt.Printf("Possui WHERE: %v\n", hasWhere)

	return hasWhere, queryModified
}

// removeOrderByAtTopLevel remove ORDER BY apenas fora de subqueries/CTEs.
func removeOrderByAtTopLevel(query string) string {
	lower := strings.ToLower(query)
	depth := 0

	for i := 0; i < len(lower)-8; i++ {
		switch lower[i] {
		case '(':
			depth++
		case ')':
			depth--
		default:
			if depth == 0 && strings.HasPrefix(lower[i:], "order by") {
				return strings.TrimSpace(query[:i])
			}
		}
	}

	return query
}

func (d PostgresDialect) BuildInsertQuery(job models.Job, records []map[string]interface{}) (string, []interface{}) {
	columns := job.Columns
	valueStrings := []string{}

	for _, record := range records {
		vals := []string{}
		for _, col := range columns {
			val := record[col]
			vals = append(vals, escapeValue(val))
		}
		valueStrings = append(valueStrings, fmt.Sprintf("(%s)", strings.Join(vals, ", ")))
	}

	// Constrói query com ON CONFLICT para evitar duplicatas
	// Assume que a primeira coluna é a chave primária
	if len(columns) > 0 {
		pkColumn := columns[0]
		query := fmt.Sprintf(`%s VALUES %s ON CONFLICT (%s) DO NOTHING`,
			job.InsertSQL,
			strings.Join(valueStrings, ", "),
			pkColumn,
		)
		return query, nil
	}

	// Fallback para INSERT simples se não houver colunas
	query := fmt.Sprintf("%s VALUES %s",
		job.InsertSQL,
		strings.Join(valueStrings, ", "),
	)

	return query, nil
}

func escapeValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return "NULL"
	case string:
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	case []byte:
		return "'" + strings.ReplaceAll(string(v), "'", "''") + "'"
	case *time.Time:
		if v == nil || v.IsZero() {
			return "NULL"
		}
		return fmt.Sprintf("'%s'", v.UTC().Format("2006-01-02 15:04:05Z07:00"))
	case time.Time:
		if v.IsZero() {
			return "NULL"
		}
		return fmt.Sprintf("'%s'", v.UTC().Format("2006-01-02 15:04:05"))
	case sql.NullTime:
		if !v.Valid || v.Time.IsZero() {
			return "NULL"
		}
		return fmt.Sprintf("'%s'", v.Time.UTC().Format("2006-01-02 15:04:05Z07:00"))
	default:
		return fmt.Sprintf("%v", v)
	}
}
