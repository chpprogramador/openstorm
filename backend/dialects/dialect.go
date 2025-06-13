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

type AccessDialect struct{}

func (d AccessDialect) FetchTotalCount(db *sql.DB, job models.Job) (int, error) {
	return 0, fmt.Errorf("COUNT(*) não suportado para MS Access")
}

func (d AccessDialect) BuildPaginatedInsertQuery(job models.Job, offset, limit int) string {
	return ""
}
