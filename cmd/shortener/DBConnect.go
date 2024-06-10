package main

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func pgConnect(connectString string) (*sql.DB, error) {
	return sql.Open("pgx", connectString)
}
