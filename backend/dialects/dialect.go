package dialects

import (
	"database/sql"
	"etl/models"
	"fmt"
	"log"
	"strings"
)

// PostgresDialect implementation

type PostgresDialect struct{}

func (d PostgresDialect) FetchBatches(db *sql.DB, job models.Job) ([][][]any, error) {
	batches := [][][]any{}
	lastRowNumber := int64(0)

	query := fmt.Sprintf(`
        SELECT t.*, ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS rn
        FROM (%s) t
    `, job.SelectSQL)

	for {
		pagedQuery := fmt.Sprintf(`
            SELECT * FROM (%s) sub
            WHERE rn > $1
            ORDER BY rn
            LIMIT $2
        `, query)

		rows, err := db.Query(pagedQuery, lastRowNumber, job.RecordsPerPage)
		if err != nil {
			return nil, err
		}

		cols, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, err
		}

		batch := [][]any{}
		maxRn := lastRowNumber

		for rows.Next() {
			vals := make([]any, len(cols))
			valPtrs := make([]any, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}

			if err := rows.Scan(valPtrs...); err != nil {
				rows.Close()
				return nil, err
			}

			// A última coluna é o rn (posição len(cols)-1)
			rn := vals[len(vals)-1].(int64)

			// Remove o rn para manter só as colunas reais
			vals = vals[:len(vals)-1]

			batch = append(batch, vals)
			if rn > maxRn {
				maxRn = rn
			}
		}

		rows.Close()

		if len(batch) == 0 {
			break
		}

		batches = append(batches, batch)
		lastRowNumber = maxRn
	}

	return batches, nil
}

func (d PostgresDialect) InsertBatch(db *sql.DB, insertSQL string, batch [][]any) error {
	if len(batch) == 0 {
		return nil
	}

	// Extrai nome da tabela e colunas do insertSQL base
	// Ex: insert into table_name (col1, col2) values
	openParen := strings.Index(insertSQL, "(")
	if openParen == -1 {
		return fmt.Errorf("insertSQL inválido: %s", insertSQL)
	}
	prefix := insertSQL[:openParen]
	colsPart := insertSQL[openParen : strings.Index(insertSQL, ")")+1]
	columns := strings.Split(strings.Trim(colsPart, "() "), ",")
	for i := range columns {
		columns[i] = strings.TrimSpace(columns[i])
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString("(" + strings.Join(columns, ", ") + ") VALUES ")

	// Monta todos os valores em linhas como: ('val1', 2, NULL)
	valLines := make([]string, len(batch))
	for i, row := range batch {
		valParts := make([]string, len(row))
		for j, val := range row {
			valParts[j] = formatValue(val)
		}
		valLines[i] = "(" + strings.Join(valParts, ", ") + ")"
	}

	sb.WriteString(strings.Join(valLines, ", "))
	sb.WriteString(";")

	query := sb.String()
	log.Printf("Executando insert de %d linhas", len(batch))

	_, err := db.Exec(query)
	return err
}

func formatValue(val any) string {
	switch v := val.(type) {
	case nil:
		return "NULL"
	case string:
		escaped := strings.ReplaceAll(v, "'", "''")
		return "'" + escaped + "'"
	case []byte:
		escaped := strings.ReplaceAll(string(v), "'", "''")
		return "'" + escaped + "'"
	case int, int64, int32, float64, float32:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		escaped := strings.ReplaceAll(fmt.Sprintf("%v", v), "'", "''")
		return "'" + escaped + "'"
	}
}

// SQLServerDialect implementation

type SQLServerDialect struct{}

func (d SQLServerDialect) FetchBatches(db *sql.DB, job models.Job) ([][][]any, error) {
	batches := [][][]any{}
	lastID := int64(0)
	for {
		query := fmt.Sprintf(`
			SELECT * FROM (
				SELECT *, ROW_NUMBER() OVER (ORDER BY id) AS RowNum FROM (%s) AS Base
			) AS Numbered
			WHERE id > ? AND RowNum <= ?`, job.SelectSQL)
		rows, err := db.Query(query, lastID, job.RecordsPerPage)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		batch := [][]any{}
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			valPtrs := make([]any, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}
			if err := rows.Scan(valPtrs...); err != nil {
				return nil, err
			}
			batch = append(batch, vals)
			lastID = vals[0].(int64)
		}
		if len(batch) == 0 {
			break
		}
		batches = append(batches, batch)
	}
	return batches, nil
}

func (d SQLServerDialect) InsertBatch(db *sql.DB, insertSQL string, batch [][]any) error {
	if len(batch) == 0 {
		return nil
	}

	openParen := strings.Index(insertSQL, "(")
	if openParen == -1 {
		return fmt.Errorf("insertSQL inválido: %s", insertSQL)
	}
	prefix := insertSQL[:openParen]
	colsPart := insertSQL[openParen : strings.Index(insertSQL, ")")+1]
	columns := strings.Split(strings.Trim(colsPart, "() "), ",")
	for i := range columns {
		columns[i] = strings.TrimSpace(columns[i])
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString("(" + strings.Join(columns, ", ") + ") VALUES ")

	valLines := make([]string, len(batch))
	for i, row := range batch {
		valParts := make([]string, len(row))
		for j, val := range row {
			valParts[j] = formatValue(val)
		}
		valLines[i] = "(" + strings.Join(valParts, ", ") + ")"
	}

	sb.WriteString(strings.Join(valLines, ", "))
	sb.WriteString(";")

	query := sb.String()
	log.Printf("Executando insert SQL Server com %d linhas", len(batch))

	_, err := db.Exec(query)
	return err
}

// MysqlDialect implementation

type MysqlDialect struct{}

func (d MysqlDialect) FetchBatches(db *sql.DB, job models.Job) ([][][]any, error) {
	batches := [][][]any{}
	lastID := int64(0)
	for {
		query := job.SelectSQL + " WHERE id > ? ORDER BY id ASC LIMIT ?"
		rows, err := db.Query(query, lastID, job.RecordsPerPage)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		batch := [][]any{}
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			valPtrs := make([]any, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}
			if err := rows.Scan(valPtrs...); err != nil {
				return nil, err
			}
			batch = append(batch, vals)
			lastID = vals[0].(int64)
		}
		if len(batch) == 0 {
			break
		}
		batches = append(batches, batch)
	}
	return batches, nil
}

func (d MysqlDialect) InsertBatch(db *sql.DB, insertSQL string, batch [][]any) error {
	if len(batch) == 0 {
		return nil
	}

	openParen := strings.Index(insertSQL, "(")
	if openParen == -1 {
		return fmt.Errorf("insertSQL inválido: %s", insertSQL)
	}
	prefix := insertSQL[:openParen]
	colsPart := insertSQL[openParen : strings.Index(insertSQL, ")")+1]
	columns := strings.Split(strings.Trim(colsPart, "() "), ",")
	for i := range columns {
		columns[i] = strings.TrimSpace(columns[i])
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString("(" + strings.Join(columns, ", ") + ") VALUES ")

	valLines := make([]string, len(batch))
	for i, row := range batch {
		valParts := make([]string, len(row))
		for j, val := range row {
			valParts[j] = formatValue(val)
		}
		valLines[i] = "(" + strings.Join(valParts, ", ") + ")"
	}

	sb.WriteString(strings.Join(valLines, ", "))
	sb.WriteString(";")

	query := sb.String()
	log.Printf("Executando insert MySQL com %d linhas", len(batch))

	_, err := db.Exec(query)
	return err
}

// AccessDialect (simples: sem paginação por ID, depende de SELECTs limitados)
type AccessDialect struct{}

func (d AccessDialect) FetchBatches(db *sql.DB, job models.Job) ([][][]any, error) {
	batches := [][][]any{}
	lastID := int64(0)
	for {
		query := fmt.Sprintf("%s WHERE id > ? ORDER BY id", job.SelectSQL)
		rows, err := db.Query(query, lastID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		batch := [][]any{}
		cols, _ := rows.Columns()
		count := 0
		for rows.Next() {
			vals := make([]any, len(cols))
			valPtrs := make([]any, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}
			if err := rows.Scan(valPtrs...); err != nil {
				return nil, err
			}
			batch = append(batch, vals)
			lastID = vals[0].(int64)
			count++
			if count >= job.RecordsPerPage {
				break
			}
		}
		if len(batch) == 0 {
			break
		}
		batches = append(batches, batch)
	}
	return batches, nil
}

func (d AccessDialect) InsertBatch(db *sql.DB, insertSQL string, batch [][]any) error {
	if len(batch) == 0 {
		return nil
	}

	openParen := strings.Index(insertSQL, "(")
	if openParen == -1 {
		return fmt.Errorf("insertSQL inválido: %s", insertSQL)
	}
	prefix := insertSQL[:openParen]
	colsPart := insertSQL[openParen : strings.Index(insertSQL, ")")+1]
	columns := strings.Split(strings.Trim(colsPart, "() "), ",")
	for i := range columns {
		columns[i] = strings.TrimSpace(columns[i])
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString("(" + strings.Join(columns, ", ") + ") VALUES ")

	valLines := make([]string, len(batch))
	for i, row := range batch {
		valParts := make([]string, len(row))
		for j, val := range row {
			valParts[j] = formatValue(val)
		}
		valLines[i] = "(" + strings.Join(valParts, ", ") + ")"
	}

	sb.WriteString(strings.Join(valLines, ", "))
	sb.WriteString(";")

	query := sb.String()
	log.Printf("Executando insert Access com %d linhas", len(batch))

	_, err := db.Exec(query)
	return err
}
