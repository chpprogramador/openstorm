package models

import "encoding/json"

type Job struct {
	ID             string   `json:"id"`
	JobName        string   `json:"jobName"`
	SelectSQL      string   `json:"selectSql"`
	InsertSQL      string   `json:"insertSql"`
	PostInsert     string   `json:"posInsertSql"`
	Columns        []string `json:"columns"`
	PrimaryKeys    []string `json:"primaryKeys"`
	RecordsPerPage int      `json:"recordsPerPage"`
	Type           string   `json:"type"`
	StopOnError    bool     `json:"stopOnError"`
	Left           int      `json:"left"`
	Top            int      `json:"top"`
}

// UnmarshalJSON aceita tanto posInsertSql (novo) quanto posInsert (legado).
func (j *Job) UnmarshalJSON(data []byte) error {
	type jobJSON struct {
		ID             string   `json:"id"`
		JobName        string   `json:"jobName"`
		SelectSQL      string   `json:"selectSql"`
		InsertSQL      string   `json:"insertSql"`
		PostInsert     string   `json:"posInsertSql"`
		LegacyInsert   string   `json:"posInsert"`
		Columns        []string `json:"columns"`
		PrimaryKeys    []string `json:"primaryKeys"`
		RecordsPerPage int      `json:"recordsPerPage"`
		Type           string   `json:"type"`
		StopOnError    bool     `json:"stopOnError"`
		Left           int      `json:"left"`
		Top            int      `json:"top"`
	}

	var aux jobJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	j.ID = aux.ID
	j.JobName = aux.JobName
	j.SelectSQL = aux.SelectSQL
	j.InsertSQL = aux.InsertSQL
	if aux.PostInsert != "" {
		j.PostInsert = aux.PostInsert
	} else {
		j.PostInsert = aux.LegacyInsert
	}
	j.Columns = aux.Columns
	j.PrimaryKeys = aux.PrimaryKeys
	j.RecordsPerPage = aux.RecordsPerPage
	j.Type = aux.Type
	j.StopOnError = aux.StopOnError
	j.Left = aux.Left
	j.Top = aux.Top

	return nil
}

// MarshalJSON mantém compatibilidade emitindo posInsertSql e posInsert.
func (j Job) MarshalJSON() ([]byte, error) {
	type jobJSON struct {
		ID             string   `json:"id"`
		JobName        string   `json:"jobName"`
		SelectSQL      string   `json:"selectSql"`
		InsertSQL      string   `json:"insertSql"`
		PostInsert     string   `json:"posInsertSql,omitempty"`
		LegacyInsert   string   `json:"posInsert,omitempty"`
		Columns        []string `json:"columns"`
		PrimaryKeys    []string `json:"primaryKeys"`
		RecordsPerPage int      `json:"recordsPerPage"`
		Type           string   `json:"type"`
		StopOnError    bool     `json:"stopOnError"`
		Left           int      `json:"left"`
		Top            int      `json:"top"`
	}

	out := jobJSON{
		ID:             j.ID,
		JobName:        j.JobName,
		SelectSQL:      j.SelectSQL,
		InsertSQL:      j.InsertSQL,
		PostInsert:     j.PostInsert,
		LegacyInsert:   j.PostInsert,
		Columns:        j.Columns,
		PrimaryKeys:    j.PrimaryKeys,
		RecordsPerPage: j.RecordsPerPage,
		Type:           j.Type,
		StopOnError:    j.StopOnError,
		Left:           j.Left,
		Top:            j.Top,
	}

	return json.Marshal(out)
}

type ValidateJobRequest struct {
	SelectSQL string `json:"selectSQL"`
	InsertSQL string `json:"insertSQL"`
	Limit     int    `json:"limit"`
	ProjectID string `json:"projectId"`
}

type ValidateJobResponse struct {
	Columns []string `json:"columns"`
	Valid   bool     `json:"valid"`
	Message string   `json:"message"`
}
