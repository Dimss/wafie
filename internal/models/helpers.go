package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type GormStringArray []string

func (a GormStringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *GormStringArray) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("unsupported type for GormStringArray: %T", value)
	}
}
