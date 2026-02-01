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

type Variable struct {
	Name        string      `json:"name"`
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
}

type VisualElement struct {
	ID            string `json:"id"`
	Type          string `json:"type"` // "rect", "circle", "line", "text"
	X             int    `json:"x"`
	Y             int    `json:"y"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	Radius        int    `json:"radius,omitempty"`
	X2            int    `json:"x2,omitempty"`
	Y2            int    `json:"y2,omitempty"`
	FillColor     string `json:"fillColor,omitempty"`
	BorderColor   string `json:"borderColor,omitempty"`
	BorderWidth   int    `json:"borderWidth,omitempty"`
	Text          string `json:"text,omitempty"`
	TextColor     string `json:"textColor,omitempty"`
	TextAlign     string `json:"textAlign,omitempty"`
	FontSize      int    `json:"fontSize,omitempty"`
	FontFamily    string `json:"fontFamily,omitempty"`
	CornerRadius  int    `json:"cornerRadius,omitempty"`
}

type Project struct {
	ID                  string          `json:"id"`
	ProjectName         string          `json:"projectName"`
	Jobs                []string        `json:"jobs"`
	Connections         []JobConnection `json:"connections"`
	SourceDatabase      DatabaseConfig  `json:"sourceDatabase"`
	DestinationDatabase DatabaseConfig  `json:"destinationDatabase"`
	Concurrency         int             `json:"concurrency"`
	Variables           []Variable      `json:"variables"`
	VisualElements      []VisualElement `json:"visualElements,omitempty"`
}
