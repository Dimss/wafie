package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type GormStringArray []string

func (a GormStringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *GormStringArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to convert value to byte slice")
	}
	return json.Unmarshal(bytes, a)
}
