// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package storage

import (
	"database/sql"
)

type CorednsRecord struct {
	ID         int64
	Zone       string
	Name       string
	Ttl        sql.NullInt32
	Content    sql.NullString
	RecordType string
}
