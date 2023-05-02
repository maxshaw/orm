package qb

type H map[string]any

type SortBy int

const (
	_ SortBy = iota
	Ascend
	Descend
)
