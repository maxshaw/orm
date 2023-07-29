package types

import (
	"fmt"
	"strings"
	"time"
)

type Time time.Time

func (t Time) IsZero() bool {
	return t.IsZero()
}

func (t *Time) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), `"`)
	nt, err := time.Parse(time.DateTime, s)
	*t = Time(nt)
	return
}

func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

func (ct *Time) String() string {
	t := time.Time(*ct)
	return fmt.Sprintf("%q", t.Format(time.DateTime))
}
