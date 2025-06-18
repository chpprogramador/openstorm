package dialects

import (
	"database/sql"
	"etl/models"
	"fmt"
	"strings"
)

type SQLDialect interface {
	FetchTotalCount(db *sql.DB, job models.Job) (int, error)
	BuildPaginatedInsertQuery(job models.Job, offset, limit int) string
	BuildInsertQuery(job models.Job, records []map[string]interface{}) (string, []interface{})
	BuildPaginatedSelectQuery(job models.Job, offset int, limit int) string
}

type PostgresDialect struct{}

func (d PostgresDialect) FetchTotalCount(db *sql.DB, job models.Job) (int, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_subquery", job.SelectSQL)
	var count int
	err := db.QueryRow(countQuery).Scan(&count)
	return count, err
}

func (d PostgresDialect) BuildPaginatedInsertQuery(job models.Job, offset, limit int) string {
	cols := strings.Join(job.Columns, ", ")
	return fmt.Sprintf(`%s
SELECT %s FROM (
    SELECT ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS rn, t.*
    FROM (
        %s
    ) t
) sub
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

func (d PostgresDialect) BuildPaginatedSelectQuery(job models.Job, offset, limit int) string {
	return fmt.Sprintf("SELECT %s FROM (%s) LIMIT %d OFFSET %d",
		strings.Join(job.Columns, ", "),
		job.SelectSQL,
		limit,
		offset,
	)
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
	default:
		return fmt.Sprintf("%v", v)
	}
}
