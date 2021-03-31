package models

import (
    "log"
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

var (
    dbt, conn *sql.DB
)

// Common pool prepare db connection
func InitDB(driverName, dataSourceName string) (*sql.DB, error) {
    conn, err := sql.Open(driverName, dataSourceName)
    log.Println("open main db conn")
    // if err != nil {
    //     log.Fatal("DB is not connected")
    // }
    // if err = conn.Ping(); err != nil {
    //     log.Fatal("DB is not responded")
    // }
    return conn, err
}
