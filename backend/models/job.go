package models

type Job struct {
	ID             string   `json:"id"`
	JobName        string   `json:"jobName"`
	SelectSQL      string   `json:"selectSql"`
	InsertSQL      string   `json:"insertSql"`
	Columns        []string `json:"columns"`
	PrimaryKeys    []string `json:"primaryKeys"`
	RecordsPerPage int      `json:"recordsPerPage"`
	Type           string   `json:"type"`
	StopOnError    bool     `json:"stopOnError"`
	Left           int      `json:"left"`
	Top            int      `json:"top"`
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
