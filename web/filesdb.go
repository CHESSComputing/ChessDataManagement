package main

// FilesDB module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

// for Go database API: http://go-database-sql.org/overview.html
// tutorial: https://golang-basic.blogspot.com/2014/06/golang-database-step-by-step-guide-on.html
// Oracle drivers:
//   _ "gopkg.in/rana/ora.v4"
//   _ "github.com/mattn/go-oci8"
// MySQL driver:
//   _ "github.com/go-sql-driver/mysql"
// SQLite driver:
//  _ "github.com/mattn/go-sqlite3"
//

import (
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

// FilesDB global variable to keep pointer to Files DB
var FilesDB *sql.DB

// InitFilesDB sets pointer to FilesDB
func InitFilesDB() (*sql.DB, error) {
	dbAttrs := strings.Split(Config.FilesDBUri, "://")
	if len(dbAttrs) != 2 {
		return nil, errors.New("Please provide proper FilesDB uri")
	}
	dbSafeAttrs := strings.Split(dbAttrs[1], "@")
	if len(dbSafeAttrs) > 1 {
		log.Printf("FilesDB: %v@%v\n", dbAttrs[0], dbSafeAttrs[1])
	} else {
		log.Printf("FilesDB: %v\n", dbAttrs)
	}
	db, err := sql.Open(dbAttrs[0], dbAttrs[1])
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(100)
	return db, err
}

// generic API to execute given statement, ideas are taken from
// http://stackoverflow.com/questions/17845619/how-to-call-the-scan-variadic-function-in-golang-using-reflection
func execute(tx *sql.Tx, stm string, args ...interface{}) ([]Record, error) {
	var records []Record

	rows, err := tx.Query(stm, args...)
	if err != nil {
		log.Printf("query %v arguments %v error %v\n", stm, args, err)
		return records, err
	}
	defer rows.Close()

	// extract columns from Rows object and create values & valuesPtrs to retrieve results
	columns, _ := rows.Columns()
	var cols []string
	count := len(columns)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	rowCount := 0

	for rows.Next() {
		if rowCount == 0 {
			// initialize value pointers
			for i := range columns {
				valuePtrs[i] = &values[i]
			}
		}
		err := rows.Scan(valuePtrs...)
		if err != nil {
			log.Printf("rows.Scan values %v error %v\n", valuePtrs, err)
			return records, err
		}
		rowCount++
		// store results into generic record (a dict)
		rec := make(Record)
		for i, col := range columns {
			if len(cols) != len(columns) {
				cols = append(cols, strings.ToLower(col))
			}
			vvv := values[i]
			switch val := vvv.(type) {
			case *sql.NullString:
				v, e := val.Value()
				if e == nil {
					rec[cols[i]] = v
				}
			case *sql.NullInt64:
				v, e := val.Value()
				if e == nil {
					rec[cols[i]] = v
				}
			case *sql.NullFloat64:
				v, e := val.Value()
				if e == nil {
					rec[cols[i]] = v
				}
			case *sql.NullBool:
				v, e := val.Value()
				if e == nil {
					rec[cols[i]] = v
				}
			default:
				//                 fmt.Printf("SQL result: %v (%T) %v (%T)\n", vvv, vvv, val, val)
				rec[cols[i]] = val
			}
			//             rec[cols[i]] = values[i]
		}
		records = append(records, rec)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Rows error %v\n", err)
		return records, err
	}
	return records, nil
}

// findDID finds dataset attributes
func findDID(stmt string, args ...interface{}) (string, error) {
	var did string
	err := FilesDB.QueryRow(stmt, args...).Scan(&did)
	if err == nil {
		return did, nil
	}
	return did, errors.New("Unable to find id")
}

// InsertFiles insert given files into FilesDB
func InsertFiles(did, dataset, path string) error {
	// look-up files for given path
	files := FindFiles(path)

	log.Printf("InsertFiles: dataset=%s did=%s", dataset, did)

	// check if we have already our dataset in DB
	dstmt := "SELECT DID FROM METADATA M JOIN DATASETS D ON M.META_ID=D.META_ID WHERE D.DATASET=? AND M.DID=?"
	DID, e := findDID(dstmt, dataset, did)
	if e == nil && DID == did {
		return nil
	}
	log.Println("proceed with insert")

	// proceed with transaction operation
	tx, err := FilesDB.Begin()
	if err != nil {
		log.Printf("ERROR: DB error %v\n", err)
		return err
	}
	defer tx.Rollback()

	// main attributes
	var stmt string
	var rec Record
	var res []Record
	create_at := time.Now().Unix()
	modify_at := time.Now().Unix()
	create_by := "MetaData server"
	modify_by := "MetaData server"

	// insert main attributes
	stmt = "INSERT INTO METADATA (DID,CREATE_AT,CREATE_BY,MODIFY_AT,MODIFY_BY) VALUES (?,?,?,?,?)"
	_, err = tx.Exec(stmt, did, create_at, create_by, modify_at, modify_by)
	if err != nil && !strings.Contains(err.Error(), "UNIQUE") {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, did, err)
		return tx.Rollback()
	}

	stmt = "SELECT META_ID FROM METADATA WHERE DID=?"
	res, err = execute(tx, stmt, did)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, did, err)
		return tx.Rollback()
	}
	rec = res[0]
	metaId := rec["meta_id"].(int64)

	// insert main attributes
	stmt = "INSERT INTO DATASETS (DATASET,META_ID,CREATE_AT,CREATE_BY,MODIFY_AT,MODIFY_BY) VALUES (?,?,?,?,?,?)"
	_, err = tx.Exec(stmt, dataset, metaId, create_at, create_by, modify_at, modify_by)
	if err != nil && !strings.Contains(err.Error(), "UNIQUE") {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, dataset, err)
		return tx.Rollback()
	}

	// select main attributes ids
	stmt = "SELECT DATASET_ID FROM DATASETS WHERE DATASET=?"
	res, err = execute(tx, stmt, dataset)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, dataset, err)
		return tx.Rollback()
	}
	rec = res[0]
	datasetId := rec["dataset_id"].(int64)

	// insert files info
	for _, fname := range files {
		stmt = "INSERT INTO FILES (FILE,META_ID,DATASET_ID,CREATE_AT,CREATE_BY,MODIFY_AT,MODIFY_BY) VALUES (?,?,?,?,?,?,?)"
		_, err = tx.Exec(stmt, fname, metaId, datasetId, create_at, create_by, modify_at, modify_by)
		if err != nil && !strings.Contains(err.Error(), "UNIQUE") {
			log.Printf("ERROR: unable to execute %s with did=%v name=%s error=%v", stmt, did, fname, err)
			return tx.Rollback()
		}
	}
	// commit whole workflow
	err = tx.Commit()
	if err != nil {
		log.Printf("ERROR: unable to commit, error=%v", err)
		return tx.Rollback()
	}
	return nil
}

// helper function to get list of files
func getFiles(did string) ([]string, error) {
	var files []string
	// proceed with transaction operation
	tx, err := FilesDB.Begin()
	if err != nil {
		log.Printf("ERROR: DB error %v\n", err)
		return files, err
	}
	defer tx.Rollback()
	// look-up files info
	//     stmt := "SELECT name FROM files WHERE meta_id=?"
	stmt := "SELECT F.FILE FROM FILES F JOIN METADATA M ON M.META_ID=F.META_ID WHERE M.DID=?"
	res, err := tx.Query(stmt, did)
	if err != nil {
		log.Printf("ERROR: unable to execute %s, error=%v", stmt, err)
		return files, tx.Rollback()
	}
	for res.Next() {
		var name string
		err = res.Scan(&name)
		if err != nil {
			log.Printf("ERROR: unable to scan error=%v", err)
			return files, tx.Rollback()
		}
		files = append(files, name)
	}
	return files, nil
}

// helper function to get list of names from a give table
func getTableNames(tname string) ([]string, error) {
	var out []string
	// proceed with transaction operation
	tx, err := FilesDB.Begin()
	if err != nil {
		log.Printf("ERROR: DB error %v\n", err)
		return out, err
	}
	defer tx.Rollback()
	stmt := "SELECT name FROM " + tname
	res, err := tx.Query(stmt)
	if err != nil {
		log.Printf("ERROR: unable to execute %s, error=%v", stmt, err)
		return out, tx.Rollback()
	}
	for res.Next() {
		var name string
		err = res.Scan(&name)
		if err != nil {
			log.Printf("ERROR: unable to scan error=%v", err)
			return out, tx.Rollback()
		}
		out = append(out, name)
	}
	return out, nil
}

// helper function to get list of datasets
func getDatasets() ([]string, error) {
	var out []string
	// proceed with transaction operation
	tx, err := FilesDB.Begin()
	if err != nil {
		log.Printf("ERROR: DB error %v\n", err)
		return out, err
	}
	defer tx.Rollback()
	// dataset is a /cycle/beamline/BTR/sample
	stmt := "SELECT D.DATASET FROM DATASETS D"
	res, err := tx.Query(stmt)
	if err != nil {
		log.Printf("ERROR: unable to execute %s, error=%v", stmt, err)
		return out, tx.Rollback()
	}
	for res.Next() {
		var dataset string
		err = res.Scan(&dataset)
		if err != nil {
			log.Printf("ERROR: unable to scan error=%v", err)
			return out, tx.Rollback()
		}
		out = append(out, dataset)
	}
	return out, nil
}
