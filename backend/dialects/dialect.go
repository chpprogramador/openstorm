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
	BuildPaginatedInsertQuery(job models.Job, offset, limit int) string
	BuildInsertQuery(job models.Job, records []map[string]interface{}) (string, []interface{})
	BuildPaginatedSelectQuery(job models.Job, offset int, limit int) string
	BuildPaginatedSelectQueryWithOrder(job models.Job, offset int, limit int, orderByColumns []string) string
}

type PostgresDialect struct{}

func (d PostgresDialect) FetchTotalCount(db *sql.DB, job models.Job) (int, error) {
	//countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_subquery", job.SelectSQL)
	//_, modifiedSQL := AnalyzeAndModifySQL(job.SelectSQL)
	//countQuery := modifiedSQL + " LIMIT 1" // Limita a 1 para otimizar a contagem

	selectRegex := regexp.MustCompile(`(?i)^SELECT\s+(.*?)\s+FROM`)
	countSQL := selectRegex.ReplaceAllString(job.SelectSQL, "SELECT COUNT(*) FROM")

	var count int
	err := db.QueryRow(countSQL).Scan(&count)
	return count, err
}

func (d PostgresDialect) BuildPaginatedInsertQuery(job models.Job, offset, limit int) string {
	//cols := strings.Join(job.Columns, ", ")
	// 	return fmt.Sprintf(`%s
	// SELECT %s FROM (
	//     SELECT ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS rn, t.*
	//     FROM (
	//         %s
	//     ) t
	// ) sub
	// WHERE rn > %d
	// ORDER BY rn
	// LIMIT %d;`,
	// 		job.InsertSQL,
	// 		cols,
	// 		job.SelectSQL,
	// 		offset,
	// 		limit,
	// 	)
	hasWhere, modifiedSQL := AnalyzeAndModifySQL(job.SelectSQL)

	//get timestamp
	time := time.Now().UnixNano()

	modifiedSQL = "CREATE sequence if not exists seq_" + fmt.Sprintf("%d", time) + "_id START 1;" + modifiedSQL

	if !hasWhere {
		modifiedSQL = fmt.Sprintf("%s where nextval('seq_"+fmt.Sprintf("%d", time)+"_id') > %d order by CTID;", modifiedSQL, offset)
	} else {
		modifiedSQL = fmt.Sprintf("%s and nextval('seq_"+fmt.Sprintf("%d", time)+"_id') > %d order by CTID;", modifiedSQL, offset)
	}

	return modifiedSQL + "DROP SEQUENCE seq_" + fmt.Sprintf("%d", time) + "_id;"
}

func (d PostgresDialect) BuildPaginatedSelectQuery(job models.Job, offset, limit int) string {
	// Usa OFFSET e LIMIT padrão do PostgreSQL para paginação simples e eficiente
	return fmt.Sprintf("%s OFFSET %d LIMIT %d", job.SelectSQL, offset, limit)
}

func (d PostgresDialect) BuildPaginatedSelectQueryWithOrder(job models.Job, offset, limit int, orderByColumns []string) string {
	// Adiciona ORDER BY explícito para garantir consistência na paginação
	orderByClause := ""
	if len(orderByColumns) > 0 {
		orderByClause = fmt.Sprintf(" ORDER BY %s", strings.Join(orderByColumns, ", "))
	} else {
		// Se não houver colunas específicas, usa CTID como fallback para garantir ordem consistente
		orderByClause = " ORDER BY CTID"
	}
	
	// Verifica se já existe ORDER BY na query original
	lowerQuery := strings.ToLower(job.SelectSQL)
	if strings.Contains(lowerQuery, "order by") {
		// Já tem ORDER BY, usa a query original
		return fmt.Sprintf("%s OFFSET %d LIMIT %d", job.SelectSQL, offset, limit)
	}
	
	return fmt.Sprintf("%s%s OFFSET %d LIMIT %d", job.SelectSQL, orderByClause, offset, limit)
}

// AnalyzeAndModifySQL detecta se a query tem WHERE e remove LIMIT e ORDER BY (simples, ignorando subqueries/CTEs).
func AnalyzeAndModifySQL(query string) (bool, string) {
	// Normaliza a query para lowercase para buscas insensíveis a case
	lowerQuery := strings.ToLower(query)

	// Detecta se há WHERE
	hasWhere := regexp.MustCompile(`\bwhere\b`).MatchString(lowerQuery)

	// Remove LIMIT (ex: LIMIT 5 ou LIMIT 10,20)
	reLimit := regexp.MustCompile(`(?i)\blimit\s+\d+(\s*,\s*\d+)?\b`)
	queryNoLimit := reLimit.ReplaceAllString(query, "")

	// Agora, remove ORDER BY manualmente
	lowerNoLimit := strings.ToLower(queryNoLimit)
	orderByIndex := strings.Index(lowerNoLimit, "order by")

	// Declara queryNoOrder no escopo externo
	var queryNoOrder string
	if orderByIndex != -1 {
		// Encontra o início do ORDER BY na query original
		orderByStart := orderByIndex
		// Remove do ORDER BY até o final
		queryNoOrder = queryNoLimit[:orderByStart]
		queryNoOrder = strings.TrimSpace(queryNoOrder)
	} else {
		queryNoOrder = queryNoLimit
	}

	// Limpa espaços extras
	queryModified := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(queryNoOrder, " "))

	fmt.Printf("Query modificada: %s\n", len([]rune(queryModified)), queryModified)
	fmt.Printf("Possui WHERE: %v\n", hasWhere)

	return hasWhere, queryModified
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

type MySQLDialect struct{}

func (d MySQLDialect) FetchTotalCount(db *sql.DB, job models.Job) (int, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_subquery", job.SelectSQL)
	var count int
	err := db.QueryRow(countQuery).Scan(&count)
	return count, err
}

func (d MySQLDialect) BuildPaginatedInsertQuery(job models.Job, offset, limit int) string {
	cols := strings.Join(job.Columns, ", ")
	// MySQL 8+ suporta ROW_NUMBER, mas para compatibilidade pode usar LIMIT/OFFSET direto
	return fmt.Sprintf(`%s
SELECT %s FROM (
    SELECT ROW_NUMBER() OVER () AS rn, t.*
    FROM (
        %s
    ) t
) AS sub
WHERE rn > %d
ORDER BY rn
LIMIT %d;`,
		job.InsertSQL,
		cols,
		job.SelectSQL,
		offset,
		limit,
	)
}

// BuildPaginatedSelectQuery constrói uma consulta paginada para MySQL
func (d MySQLDialect) BuildPaginatedSelectQuery(job models.Job, offset int, limit int) string {
	return fmt.Sprintf("SELECT %s FROM (%s) AS sub LIMIT %d OFFSET %d",
		strings.Join(job.Columns, ", "),
		job.SelectSQL,
		limit,
		offset,
	)
}

// BuildPaginatedSelectQueryWithOrder constrói uma consulta paginada com ordenação para MySQL
func (d MySQLDialect) BuildPaginatedSelectQueryWithOrder(job models.Job, offset int, limit int, orderByColumns []string) string {
	orderByClause := ""
	if len(orderByColumns) > 0 {
		orderByClause = fmt.Sprintf(" ORDER BY %s", strings.Join(orderByColumns, ", "))
	}
	
	// Verifica se já existe ORDER BY na query original
	lowerQuery := strings.ToLower(job.SelectSQL)
	if strings.Contains(lowerQuery, "order by") {
		// Já tem ORDER BY, usa a query original
		return fmt.Sprintf("SELECT %s FROM (%s) AS sub LIMIT %d OFFSET %d",
			strings.Join(job.Columns, ", "),
			job.SelectSQL,
			limit,
			offset,
		)
	}
	
	return fmt.Sprintf("SELECT %s FROM (%s) AS sub%s LIMIT %d OFFSET %d",
		strings.Join(job.Columns, ", "),
		job.SelectSQL,
		orderByClause,
		limit,
		offset,
	)
}

// BuildInsertQuery constrói uma consulta de inserção para MySQL
func (d MySQLDialect) BuildInsertQuery(job models.Job, records []map[string]interface{}) (string, []interface{}) {
	columns := job.Columns
	valueStrings := []string{}
	var args []interface{}

	for _, record := range records {
		vals := []string{}
		for _, col := range columns {
			val := record[col]
			vals = append(vals, escapeValue(val))
			args = append(args, val) // Adiciona o valor para os parâmetros
		}
		valueStrings = append(valueStrings, fmt.Sprintf("(%s)", strings.Join(vals, ", ")))
	}

	// Constrói query com ON DUPLICATE KEY UPDATE para evitar duplicatas
	// Assume que a primeira coluna é a chave primária
	if len(columns) > 0 {
		pkColumn := columns[0]
		query := fmt.Sprintf(`%s VALUES %s ON DUPLICATE KEY UPDATE %s = VALUES(%s)`,
			job.InsertSQL,
			strings.Join(valueStrings, ", "),
			pkColumn, pkColumn,
		)
		return query, args
	}

	// Fallback para INSERT simples se não houver colunas
	query := fmt.Sprintf("%s VALUES %s",
		job.InsertSQL,
		strings.Join(valueStrings, ", "),
	)

	return query, args
}

type SQLServerDialect struct{}

func (d SQLServerDialect) FetchTotalCount(db *sql.DB, job models.Job) (int, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_subquery", job.SelectSQL)
	var count int
	err := db.QueryRow(countQuery).Scan(&count)
	return count, err
}

func (d SQLServerDialect) BuildPaginatedInsertQuery(job models.Job, offset, limit int) string {
	cols := strings.Join(job.Columns, ", ")
	return fmt.Sprintf(`%s
SELECT %s FROM (
    SELECT ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS rn, t.*
    FROM (
        %s
    ) t
) sub
WHERE rn > %d AND rn <= %d
ORDER BY rn;`,
		job.InsertSQL,
		cols,
		job.SelectSQL,
		offset,
		offset+limit,
	)
}

func (d SQLServerDialect) BuildPaginatedSelectQuery(job models.Job, offset int, limit int) string {
	return fmt.Sprintf("SELECT %s FROM (%s) AS sub WHERE rn > %d AND rn <= %d",
		strings.Join(job.Columns, ", "),
		job.SelectSQL,
		offset,
		offset+limit,
	)
}

func (d SQLServerDialect) BuildPaginatedSelectQueryWithOrder(job models.Job, offset int, limit int, orderByColumns []string) string {
	orderByClause := ""
	if len(orderByColumns) > 0 {
		orderByClause = fmt.Sprintf(" ORDER BY %s", strings.Join(orderByColumns, ", "))
	}
	
	// Verifica se já existe ORDER BY na query original
	lowerQuery := strings.ToLower(job.SelectSQL)
	if strings.Contains(lowerQuery, "order by") {
		// Já tem ORDER BY, usa a query original
		return fmt.Sprintf("SELECT %s FROM (%s) AS sub WHERE rn > %d AND rn <= %d",
			strings.Join(job.Columns, ", "),
			job.SelectSQL,
			offset,
			offset+limit,
		)
	}
	
	return fmt.Sprintf("SELECT %s FROM (%s) AS sub%s WHERE rn > %d AND rn <= %d",
		strings.Join(job.Columns, ", "),
		job.SelectSQL,
		orderByClause,
		offset,
		offset+limit,
	)
}

func (d SQLServerDialect) BuildInsertQuery(job models.Job, records []map[string]interface{}) (string, []interface{}) {
	columns := job.Columns
	valueStrings := []string{}
	var args []interface{}

	for _, record := range records {
		vals := []string{}
		for _, col := range columns {
			val := record[col]
			vals = append(vals, escapeValue(val))
			args = append(args, val) // Adiciona o valor para os parâmetros
		}
		valueStrings = append(valueStrings, fmt.Sprintf("(%s)", strings.Join(vals, ", ")))
	}

	query := fmt.Sprintf("%s VALUES %s",
		job.InsertSQL,
		strings.Join(valueStrings, ", "),
	)

	return query, args
}

type AccessDialect struct{}

func (d AccessDialect) FetchTotalCount(db *sql.DB, job models.Job) (int, error) {
	return 0, fmt.Errorf("COUNT(*) não suportado para MS Access")
}

func (d AccessDialect) BuildPaginatedInsertQuery(job models.Job, offset, limit int) string {
	return ""
}

func (d AccessDialect) BuildPaginatedSelectQuery(job models.Job, offset int, limit int) string {
	return fmt.Sprintf("SELECT TOP %d %s FROM (%s) AS sub WHERE rn > %d",
		limit,
		strings.Join(job.Columns, ", "),
		job.SelectSQL,
		offset,
	)
}

func (d AccessDialect) BuildPaginatedSelectQueryWithOrder(job models.Job, offset int, limit int, orderByColumns []string) string {
	orderByClause := ""
	if len(orderByColumns) > 0 {
		orderByClause = fmt.Sprintf(" ORDER BY %s", strings.Join(orderByColumns, ", "))
	}
	
	// Verifica se já existe ORDER BY na query original
	lowerQuery := strings.ToLower(job.SelectSQL)
	if strings.Contains(lowerQuery, "order by") {
		// Já tem ORDER BY, usa a query original
		return fmt.Sprintf("SELECT TOP %d %s FROM (%s) AS sub WHERE rn > %d",
			limit,
			strings.Join(job.Columns, ", "),
			job.SelectSQL,
			offset,
		)
	}
	
	return fmt.Sprintf("SELECT TOP %d %s FROM (%s) AS sub%s WHERE rn > %d",
		limit,
		strings.Join(job.Columns, ", "),
		job.SelectSQL,
		orderByClause,
		offset,
	)
}

// BuildInsertQuery constrói uma consulta de inserção para Access
func (d AccessDialect) BuildInsertQuery(job models.Job, records []map[string]interface{}) (string, []interface{}) {
	columns := job.Columns
	valueStrings := []string{}
	var args []interface{}

	for _, record := range records {
		vals := []string{}
		for _, col := range columns {
			val := record[col]
			vals = append(vals, escapeValue(val))
			args = append(args, val) // Adiciona o valor para os parâmetros
		}
		valueStrings = append(valueStrings, fmt.Sprintf("(%s)", strings.Join(vals, ", ")))
	}

	query := fmt.Sprintf("%s VALUES %s",
		job.InsertSQL,
		strings.Join(valueStrings, ", "),
	)

	return query, args
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
