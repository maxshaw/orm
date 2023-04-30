package orm

type ValidationError struct {
	Field, Msg string
	Underlying error
}

func (e *ValidationError) Error() string {
	return e.Msg
}
