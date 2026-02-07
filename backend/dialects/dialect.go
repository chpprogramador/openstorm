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
	hashExpr := fmt.Sprintf("mod(abs(hashtextextended(%s, 0)), %d) = %d",
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
	hasWhere := hasWhereAtTopLevel(lowerQuery)

	// Remove LIMIT apenas no nível principal
	queryNoLimit := removeLimitAtTopLevel(query, lowerQuery)

	// Remove ORDER BY somente no nível principal
	queryNoOrder := removeOrderByAtTopLevel(queryNoLimit)

	// Limpa espaços extras
	queryModified := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(queryNoOrder, " "))

	fmt.Printf("Query modificada: %d caracteres: %s\n", len([]rune(queryModified)), queryModified)
	fmt.Printf("Possui WHERE: %v\n", hasWhere)

	return hasWhere, queryModified
}

// hasWhereAtTopLevel detecta WHERE apenas no nível principal (fora de subqueries/CTEs e strings).
func hasWhereAtTopLevel(lowerQuery string) bool {
	depth := 0
	inSingle := false
	for i := 0; i < len(lowerQuery); i++ {
		ch := lowerQuery[i]
		if ch == '\'' {
			if inSingle && i+1 < len(lowerQuery) && lowerQuery[i+1] == '\'' {
				i++
				continue
			}
			inSingle = !inSingle
			continue
		}
		if inSingle {
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && i+5 <= len(lowerQuery) && lowerQuery[i:i+5] == "where" {
			prev := i == 0 || !isIdentChar(lowerQuery[i-1])
			next := i+5 == len(lowerQuery) || !isIdentChar(lowerQuery[i+5])
			if prev && next {
				return true
			}
		}
	}
	return false
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}

// removeLimitAtTopLevel remove LIMIT apenas fora de subqueries/CTEs e strings.
func removeLimitAtTopLevel(query, lower string) string {
	depth := 0
	inSingle := false
	inDouble := false

	for i := 0; i < len(lower)-5; i++ {
		ch := lower[i]

		if inSingle {
			if ch == '\'' {
				if i+1 < len(lower) && lower[i+1] == '\'' {
					i++
					continue
				}
				inSingle = false
			}
			continue
		}
		if inDouble {
			if ch == '"' {
				if i+1 < len(lower) && lower[i+1] == '"' {
					i++
					continue
				}
				inDouble = false
			}
			continue
		}

		switch ch {
		case '\'':
			inSingle = true
			continue
		case '"':
			inDouble = true
			continue
		case '(':
			depth++
			continue
		case ')':
			if depth > 0 {
				depth--
			}
			continue
		}

		if depth == 0 && strings.HasPrefix(lower[i:], "limit") {
			prev := i == 0 || !isIdentChar(lower[i-1])
			next := i+5 == len(lower) || !isIdentChar(lower[i+5])
			if prev && next {
				end := findLimitClauseEnd(lower, i+5)
				trimmed := strings.TrimSpace(query[:i] + " " + query[end:])
				return trimmed
			}
		}
	}

	return query
}

func findLimitClauseEnd(lower string, start int) int {
	i := skipSpaces(lower, start)
	if hasTokenAt(lower, i, "all") {
		i += 3
		return skipSpaces(lower, i)
	}

	i = skipNumberOrParameter(lower, i)
	i = skipSpaces(lower, i)

	if i < len(lower) && lower[i] == ',' {
		i++
		i = skipSpaces(lower, i)
		i = skipNumberOrParameter(lower, i)
		return skipSpaces(lower, i)
	}

	if hasTokenAt(lower, i, "offset") {
		i += 6
		i = skipSpaces(lower, i)
		i = skipNumberOrParameter(lower, i)
		i = skipSpaces(lower, i)
	}

	if hasTokenAt(lower, i, "fetch") {
		i += 5
		i = skipSpaces(lower, i)
		if hasTokenAt(lower, i, "first") {
			i += 5
		} else if hasTokenAt(lower, i, "next") {
			i += 4
		}
		i = skipSpaces(lower, i)
		i = skipNumberOrParameter(lower, i)
		i = skipSpaces(lower, i)
		if hasTokenAt(lower, i, "row") {
			i += 3
		} else if hasTokenAt(lower, i, "rows") {
			i += 4
		}
		i = skipSpaces(lower, i)
		if hasTokenAt(lower, i, "only") {
			i += 4
		}
		i = skipSpaces(lower, i)
	}

	return i
}

func skipSpaces(lower string, i int) int {
	for i < len(lower) {
		switch lower[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

func skipNumberOrParameter(lower string, i int) int {
	if i >= len(lower) {
		return i
	}
	if lower[i] == '$' {
		i++
		for i < len(lower) && lower[i] >= '0' && lower[i] <= '9' {
			i++
		}
		return i
	}
	if lower[i] == '?' {
		return i + 1
	}
	for i < len(lower) && lower[i] >= '0' && lower[i] <= '9' {
		i++
	}
	return i
}

func hasTokenAt(lower string, i int, token string) bool {
	if i+len(token) > len(lower) || !strings.HasPrefix(lower[i:], token) {
		return false
	}
	prev := i == 0 || !isIdentChar(lower[i-1])
	next := i+len(token) == len(lower) || !isIdentChar(lower[i+len(token)])
	return prev && next
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
		query := fmt.Sprintf(`%s VALUES %s`,
			job.InsertSQL,
			strings.Join(valueStrings, ", "),
		)
		return appendPostInsert(query, job.PostInsert), nil
	}

	// Fallback para INSERT simples se não houver colunas
	query := fmt.Sprintf("%s VALUES %s",
		job.InsertSQL,
		strings.Join(valueStrings, ", "),
	)

	return appendPostInsert(query, job.PostInsert), nil
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

func appendPostInsert(baseSQL, postInsert string) string {
	post := strings.TrimSpace(postInsert)
	if post == "" {
		return baseSQL
	}
	base := strings.TrimRight(baseSQL, " \t\r\n;")
	post = strings.TrimLeft(post, "; \t\r\n")

	merged := base + "\n" + post
	merged = strings.TrimRight(merged, " \t\r\n;")
	return merged + ";"
}
