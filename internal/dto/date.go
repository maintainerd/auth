package dto

import (
	"encoding/json"
	"time"
)

type Date struct {
	time.Time
}

const dateLayout = "2006-01-02"

func (d *Date) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == `null` || s == `""` {
		d.Time = time.Time{}
		return nil
	}

	parsed, err := time.Parse(`"`+dateLayout+`"`, s)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte(`null`), nil
	}
	return json.Marshal(d.Time.Format(dateLayout))
}
