package types

import (
	"database/sql/driver"
	"encoding/json"
)

type JSON[T any] struct {
	bytes []byte
	value *T
}

func (j *JSON[T]) Get() *T {
	return j.value
}

func (j *JSON[T]) UnmarshalJSON(b []byte) (err error) {
	if b == nil {
		j.bytes = nil
		return nil
	}

	var v T
	if err = json.Unmarshal(b, &v); err != nil {
		j.bytes = nil
	} else {
		var dst = make([]byte, len(b))
		_ = copy(dst, b)
		j.bytes, j.value = dst, &v
	}

	return
}

func (j JSON[T]) MarshalJSON() ([]byte, error) {
	if j.bytes == nil {
		return []byte("null"), nil
	}
	return j.bytes, nil
}

func (j *JSON[T]) Scan(value any) error {
	if b, ok := value.([]byte); ok {
		return j.UnmarshalJSON(b)
	}
	j.bytes, j.value = nil, nil
	return nil
}

func (j JSON[T]) Value() (driver.Value, error) {
	return j.bytes, nil
}
