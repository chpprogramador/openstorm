package models

type DatabaseConfig struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type Project struct {
	ID                  string         `json:"id"`
	ProjectName         string         `json:"projectName"`
	Jobs                []string       `json:"jobs"`
	SourceDatabase      DatabaseConfig `json:"sourceDatabase"`
	DestinationDatabase DatabaseConfig `json:"destinationDatabase"`
}
