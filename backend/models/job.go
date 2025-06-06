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
