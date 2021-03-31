package models

import (
    "fmt"
    "log"
    "time"
    // "strconv"
    "database/sql"
)

var (
    PRODMODE bool
)

type(
    Forms struct {
        Id_forms int
        Name     string
        Test_uri string
        Live_uri string
    }
    VarcharColumn struct {
        Table string
        Column string
    }
)
/*
sqlc:
SELECT count(link_rewrite) FROM ps_cms_lang WHERE TRIM(link_rewrite) REGEXP '[^.0-9a-z_-]|[0-9]{2,}';
sql:
SELECT DISTINCT link_rewrite FROM ps_cms_lang WHERE TRIM(link_rewrite) NOT REGEXP '[^.0-9a-z_-]|[0-9]{2,}';
*/
func ExtractValidPathes(db *sql.DB, table, col string) ([]string, error) {
    var count int
    regexp := " REGEXP '[^.0-9a-z_-]|[0-9]{2,}'"
    sql  := "SELECT DISTINCT t.`" + col + "` FROM " + table + " t WHERE TRIM(t.`" + col + "`) NOT "+ regexp 
    sqlc := "SELECT COUNT(`" + col + "`) FROM " + table + " t WHERE TRIM(t.`" + col + "`)"+ regexp
    log.Println("SQL REGEXPS sqlc:", sqlc)
    log.Println("SQL REGEXPS sql:", sql)
    err := db.QueryRow(sqlc).Scan(&count)
    switch {    
        case err != nil:
            log.Fatal(err)
        default:
            if !PRODMODE {
                fmt.Println("SQL COUNT ExtractValidPathes():", count)
            }
    }
    // defer rows.Close()
    var pathes []string
    if count < 5 {
        rows, err := db.Query(sql)
        if err != nil {
            return nil, err
        }
        defer rows.Close()

        var item string
        for rows.Next() {
            err := rows.Scan(&item)
            if err != nil {
                return nil, err
            }
            pathes = append(pathes, item)
        }
        // -if !rows.NextResultSet() {
        //     log.Fatalf("expected more result sets: %v", rows.Err())
        // }
        if err = rows.Err(); err != nil {
            return nil, err
        }
    }

    return pathes, nil
}

func GetListAllDbTables(conn *sql.DB/*, filterText string*/) ([]string, error) {
    rows, err := conn.Query("SHOW TABLES")
    if err != nil {
        log.Panic(err)
    }
    defer rows.Close()
    var table string
    var tabList []string
    for rows.Next() {
        err := rows.Scan(&table)
        if err != nil {
            log.Panic(err)
        }
        tabList = append(tabList, table)
        if !PRODMODE {
            fmt.Println("GATHERED:", table)
        }
    }
    if err = rows.Err(); err != nil {
        return nil, err
    }
    return tabList, nil
}

// Check current reserched links and forms:
func ShowResearchedLinks(db *sql.DB) ([]*Forms, error) {
    sql:= "SELECT f.id_forms, f.live_uri FROM forms f WHERE f.live_uri IS NOT NULL ORDER BY f.id_forms DESC"
    if !PRODMODE {
        fmt.Println("SQL showResearchedLinks():", sql)
    }
    rows, err := db.Query(sql)
    if err != nil {
        log.Panic(err)
    }
    defer rows.Close()

    forms := make([]*Forms, 0)
    for rows.Next() {
        f:= new(Forms)
        rows.Scan(&f.Id_forms, &f.Live_uri)
        forms = append(forms, f)
    }
    if err = rows.Err(); err != nil {
        return nil, err
    }
    return forms, nil
}
  

// Gather all appropriate columns with varchar type in target database
func GatherAllVarcharTablesColumns(db *sql.DB, dbName string) ([]VarcharColumn, error) {
    sql := "SELECT TABLE_NAME, COLUMN_NAME FROM columns WHERE TABLE_SCHEMA = 'dldev' AND DATA_TYPE like '%char' ORDER BY 1;"
    rows, err := db.Query(sql)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var columns []VarcharColumn
    for rows.Next() {
        varCol := VarcharColumn{}
        err := rows.Scan(&varCol.Table, &varCol.Column)
        if err != nil {
            return nil, err
        }
        columns = append(columns, varCol)
    }
    if err = rows.Err(); err != nil {
        return nil, err
    }
    return columns, nil
}

// Insert detected forms on the link:
func InsertDetectedForms(conn *sql.DB, f Forms) (int64, error) {
    if !PRODMODE {
        fmt.Println("TRY UPDATE FormInfo:",f.Id_forms, f.Name, f.Test_uri, f.Live_uri)
    }
    sql:= "UPDATE forms f SET f.live_uri=? WHERE f.id_forms=?"
    stmt, err := conn.Prepare(sql)
    if err != nil {
        log.Fatal(err)
    }

    res, err := stmt.Exec(
        f.Live_uri, f.Id_forms,
    )
    if err != nil {
        log.Fatal(err)
    }

    affected, err := res.RowsAffected()
    if err != nil {
        log.Fatal(err)
    }

    if !PRODMODE {
        fmt.Println("UPDATED in DB:", affected, "Id_forms:",f.Id_forms, f.Live_uri)
    }
    return affected, nil
}

// Get descript of the column on manner DESC the_some_table;
// with output by print version. From: 
// https://stackoverflow.com/questions/47662614/http-responsewriter-write-with-interface
func DescriptTable(db *sql.DB, tab string) error {
    rows, err := db.Query("DESC "+tab)
    if err != nil {
        return err
    }
    defer rows.Close()
    cols, err := rows.Columns()
    if err != nil {
        return err
    }
    if cols == nil {
        return nil
    }

    // Make header for description:
    vals := make([]interface{}, len(cols))
    for i := 0; i < len(cols); i++ {
        vals[i] = new(interface{})
        if i != 0 {
            fmt.Print("\t\t")
        }
        fmt.Print(cols[i])
    }
    fmt.Println()

    // Make description table:
    for rows.Next() {
        err = rows.Scan(vals...)
        if err != nil {
            fmt.Println(err)
            continue
        }
        for i := 0; i < len(vals); i++ {
            if i != 0 {
                fmt.Print("\t\t")
            }
            printRawValue(vals[i].(*interface{}))
        }
        fmt.Println()

    }
    if rows.Err() != nil {
        return rows.Err()
    }
    return nil
}

func printRawValue(pval *interface{}) {
    switch v := (*pval).(type) {
    case nil:
        fmt.Print("NULL")
    case bool:
        if v {
            fmt.Print("1")
        } else {
            fmt.Print("0")
        }
    case []byte:
        fmt.Print(string(v))
    case time.Time:
        fmt.Print(v.Format("2006-01-02 15:04:05.999"))
    default:
        fmt.Print(v)
    }
}


