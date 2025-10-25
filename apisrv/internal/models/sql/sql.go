package sql

import "embed"

// Embed static web assets
//
//go:embed *.sql
var sql embed.FS

func Triggers() (string, error) {
	rawSqls, err := sql.ReadFile("triggers.sql")
	if err != nil {
		return "", err
	}
	return string(rawSqls), nil
}
