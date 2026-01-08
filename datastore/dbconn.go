package datastore

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// NewDB takes arguments for db type and conn string and returns a DatabaseConnectionResult
func NewDB(dbtype string, connstr string) (*sql.DB, error) {
	db, openError := sql.Open(dbtype, connstr)

	if pingError := db.Ping(); pingError != nil {
		return &sql.DB{}, fmt.Errorf("could not establish connection with database -> %v", pingError)
	}

	if openError != nil {
		return &sql.DB{}, fmt.Errorf("error opening connection -> %v", openError)
	}

	return db, nil
}

// BuildDBConnStr builds a PostgreSQL connection string
func BuildDBConnStr(password, user, dbname, sslmode string) string {
	return fmt.Sprintf("postgres://%s:%s@localhost/%s?sslmode=%s", user, password, dbname, sslmode)
}
