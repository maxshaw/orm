package qb

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JSONValue[T any] struct {
	raw json.RawMessage
	val *T

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

// Scan implements the Scanner interface.
func (j *JSONValue[T]) Scan(value any) error {
	var err error
	if value != nil {
		if data, ok := value.([]byte); ok {
			if err = j.UnmarshalJSON(data); err == nil {
				return nil
			}
		}
	}

	j.raw, j.val, j.Valid = nil, nil, false
	return err
}

// Value implements the driver Valuer interface.
func (j JSONValue[T]) Value() (driver.Value, error) {
	if !j.Valid {
		return nil, nil
	}
	return j.raw, nil
}

// MarshalJSON returns m as the JSON encoding of m.
func (j JSONValue[T]) MarshalJSON() ([]byte, error) {
	if j.raw == nil {
		return []byte("null"), nil
	}
	return j.raw, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (j *JSONValue[T]) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSONValue[T]: UnmarshalJSON on nil pointer")
	}

	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		j.raw, j.val, j.Valid = nil, nil, false
		return nil
	}

	j.raw, j.val, j.Valid = data, &val, true
	return nil
}
