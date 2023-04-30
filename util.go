package orm

func Unwrap[T any](v *T) T {
	var t T
	if v == nil {
		return t
	}
	return *v
}

func Ptr[T any](v T) *T {
	return &v
}
