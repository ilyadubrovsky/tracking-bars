package domain

type ProgressTable struct {
	BarsCredentialsID string
	Disciplines       []Discipline
}

type Discipline struct {
	Name          string         `json:"name"`
	ControlEvents []ControlEvent `json:"control_events"`
}

type ControlEvent struct {
	Name  string `json:"name"`
	Grade string `json:"grade"`
}
