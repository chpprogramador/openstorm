package models

type DatabaseConfig struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type JobConnection struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Project struct {
	ID                  string          `json:"id"`
	ProjectName         string          `json:"projectName"`
	Jobs                []string        `json:"jobs"`
	Connections         []JobConnection `json:"connections"`
	SourceDatabase      DatabaseConfig  `json:"sourceDatabase"`
	DestinationDatabase DatabaseConfig  `json:"destinationDatabase"`
	Concurrency         int             `json:"concurrency"`
}
