package qb

import (
	"database/sql/driver"
	"encoding/json"
)

type JSONValue[T any] struct {
	raw   []byte
	val   *T
	Valid bool
}

func NewJSONValue[T any](v T) *JSONValue[T] {
	j := &JSONValue[T]{}
	raw, err := json.Marshal(v)
	if err != nil {
		j.raw, j.val, j.Valid = nil, nil, false
	} else {
		j.raw, j.val, j.Valid = raw, &v, true
	}
	return j
}

func (j JSONValue[T]) Val() T {
	if j.val == nil {
		var t T
		return t
	}
	return *j.val
}

func (j *JSONValue[T]) unpack(b []byte) (err error) {
	if b != nil {
		var v T
		if err = json.Unmarshal(b, &v); err == nil {
			var (
				bc  = len(b)
				raw = make([]byte, bc)
			)
			if n := copy(raw, b); n == bc {
				j.raw, j.val, j.Valid = raw, &v, true
				return nil
			}
		}
	}
	j.raw, j.val, j.Valid = nil, nil, false
	return
}

// Scan implements the Scanner interface.
func (j *JSONValue[T]) Scan(value any) error {
	if b, ok := value.([]byte); ok {
		return j.unpack(b)
	}
	j.raw, j.val, j.Valid = nil, nil, false
	return nil
}

// Value implements the driver Valuer interface.
func (j JSONValue[T]) Value() (driver.Value, error) {
	if j.Valid {
		return j.raw, nil
	}
	return nil, nil
}

// MarshalJSON returns m as the JSON encoding of m.
func (j JSONValue[T]) MarshalJSON() ([]byte, error) {
	if j.raw == nil {
		return []byte("null"), nil
	}
	return j.raw, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (j *JSONValue[T]) UnmarshalJSON(b []byte) error {
	return j.unpack(b)
}
