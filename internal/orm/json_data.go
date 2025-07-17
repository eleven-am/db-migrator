package orm

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONData struct {
	json.RawMessage
}

func (j *JSONData) Unmarshal(v interface{}) error {
	if len(j.RawMessage) == 0 {
		return nil
	}
	return json.Unmarshal(j.RawMessage, v)
}

func (j *JSONData) Marshal(v interface{}) error {
	if v == nil {
		j.RawMessage = nil
		return nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	j.RawMessage = data
	return nil
}

func (j *JSONData) Scan(value interface{}) error {
	if value == nil {
		j.RawMessage = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			j.RawMessage = nil
			return nil
		}
		j.RawMessage = json.RawMessage(v)
	case string:
		if v == "" {
			j.RawMessage = nil
			return nil
		}
		j.RawMessage = json.RawMessage(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONData", value)
	}

	return nil
}

func (j JSONData) Value() (driver.Value, error) {
	if len(j.RawMessage) == 0 {
		return nil, nil
	}
	return []byte(j.RawMessage), nil
}

func (j JSONData) String() string {
	if len(j.RawMessage) == 0 {
		return "null"
	}
	return string(j.RawMessage)
}
