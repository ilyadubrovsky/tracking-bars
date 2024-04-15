package domain

type GradeChange struct {
	UserID       int64
	Discipline   string
	ControlEvent string
	OldGrade     string
	NewGrade     string
}
