package models

type Job struct {
	ID             string `json:"id"`
	JobName        string `json:"jobName"`
	SelectSQL      string `json:"selectSql"`
	InsertSQL      string `json:"insertSql"`
	RecordsPerPage int    `json:"recordsPerPage"`
	Left           int    `json:"left"`
	Top            int    `json:"top"`
}

type ValidateJobRequest struct {
	SelectSQL string `json:"selectSQL"`
	InsertSQL string `json:"insertSQL"`
	Limit     int    `json:"limit"`
}

type ValidateJobResponse struct {
	Columns []string                 `json:"columns"`
	Preview []map[string]interface{} `json:"preview"`
	Valid   bool                     `json:"valid"`
	Message string                   `json:"message"`
}
