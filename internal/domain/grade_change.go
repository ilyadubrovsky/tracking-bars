package domain

type GradeChange struct {
	UserID       int64  `json:"user_id"`
	Discipline   string `json:"discipline"`
	ControlEvent string `json:"control_event"`
	OldGrade     string `json:"old_grade"`
	NewGrade     string `json:"new_grade"`
}
