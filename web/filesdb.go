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
	"strings"
	"time"
    "fmt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	logs "github.com/sirupsen/logrus"
)

// global variable to keep pointer to FilesDB
var FilesDB *sql.DB

// InitFilesDB sets pointer to FilesDB
func InitFilesDB() (*sql.DB, error) {
	dbAttrs := strings.Split(Config.FilesDBUri, "://")
	if len(dbAttrs) != 2 {
		return nil, errors.New("Please provide proper FilesDB uri")
	}
	dbSafeAttrs := strings.Split(dbAttrs[1], "@")
	logs.WithFields(logs.Fields{
		"Driver":  dbAttrs[0],
		"FilesDB": dbSafeAttrs[1],
	}).Info("FilesDB")
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
		logs.WithFields(logs.Fields{
			"Statement": stm,
			"Arguments": args,
			"Error":     err,
		}).Error("query error")
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
			for i, _ := range columns {
				valuePtrs[i] = &values[i]
			}
		}
		err := rows.Scan(valuePtrs...)
		if err != nil {
			logs.WithFields(logs.Fields{
				"Destination": valuePtrs,
				"Error":       err,
			}).Error("rows.Scan error")
			return records, err
		}
		rowCount += 1
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
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("Rows")
		return records, err
	}
	return records, nil
}

// FindId finds dataset attributes
func FindId(stmt string, args ...interface{}) (int64, error) {
	var rid int64
	err := FilesDB.QueryRow(stmt, args...).Scan(&rid)
	if err == nil {
		return rid, nil
	}
	return -1, errors.New("Unable to find id")
}

// InsertFiles insert given files into FilesDB
func InsertFiles(did int64, experiment, processing, tier, path string) error {
	// look-up files for given path
	files := FindFiles(path)

	// check if we have already our dataset in DB
	dstmt := "SELECT dataset_id FROM datasets JOIN tiers ON datasets.tier_id=tiers.tier_id JOIN processing ON datasets.processing_id=processing.processing_id JOIN experiments ON datasets.experiment_id=experiments.experiment_id WHERE experiments.name=? and processing.name=? and tiers.name=?"
	datasetId, e := FindId(dstmt, experiment, processing, tier)
	if e == nil && datasetId == did {
		return nil
	}

	// proceed with transaction operation
	tx, err := FilesDB.Begin()
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("DB error")
		return err
	}
	defer tx.Rollback()

	var res []Record
	var stmt string
	// insert main attributes
	stmt = "INSERT INTO tiers (name) VALUES (?)"
	_, err = tx.Exec(stmt, tier)
	if err != nil {
		return tx.Rollback()
	}
	stmt = "INSERT INTO processing (name) VALUES (?)"
	_, err = tx.Exec(stmt, processing)
	if err != nil {
		return tx.Rollback()
	}
	stmt = "INSERT INTO experiments (name) VALUES (?)"
	_, err = tx.Exec(stmt, experiment)
	if err != nil {
		return tx.Rollback()
	}

	// select main attributes ids
	var rec Record
	stmt = "SELECT experiment_id FROM experiments WHERE name=?"
	res, err = execute(tx, stmt, experiment)
	if err != nil {
		return tx.Rollback()
	}
	rec = res[0]
	eid := rec["experiment_id"].(int64)

	stmt = "SELECT processing_id FROM processing WHERE name=?"
	res, err = execute(tx, stmt, processing)
	if err != nil {
		return tx.Rollback()
	}
	rec = res[0]
	pid := rec["processing_id"].(int64)

	stmt = "SELECT tier_id FROM tiers WHERE name=?"
	res, err = execute(tx, stmt, tier)
	if err != nil {
		return tx.Rollback()
	}
	rec = res[0]
	tid := rec["tier_id"].(int64)

	// insert data into datasets table
	tstamp := time.Now()
	stmt = "INSERT INTO datasets (dataset_id,experiment_id,processing_id,tier_id,tstamp) VALUES (?, ?, ?, ?, ?)"
	_, err = tx.Exec(stmt, did, eid, pid, tid, tstamp)
	if err != nil {
		return tx.Rollback()
	}

	// insert files info
	for _, name := range files {
		stmt = "INSERT INTO files (dataset_id,name) VALUES (?,?)"
		_, err = tx.Exec(stmt, did, name)
		if err != nil {
			return tx.Rollback()
		}
	}
	// commit whole workflow
	err = tx.Commit()
	if err != nil {
		return tx.Rollback()
	}
	return nil
}

// helper function to get list of files
func getFiles(did int64) ([]string, error) {
	var files []string
	// proceed with transaction operation
	tx, err := FilesDB.Begin()
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("DB error")
		return files, err
	}
	defer tx.Rollback()
	// look-up files info
	stmt := "SELECT name FROM files WHERE dataset_id=?"
//	res, err := execute(tx, stmt, did)
    res, err := tx.Query(stmt, did)
	if err != nil {
		return files, tx.Rollback()
	}
    for res.Next() {
        var name string
        err = res.Scan(&name)
        if err != nil {
            return files, tx.Rollback()
        }
        fmt.Println("file", name)
        files = append(files, name)
    }
    
//	for _, r := range res {
//		files = append(files, r["name"].(string))
//	}
	return files, nil
}
