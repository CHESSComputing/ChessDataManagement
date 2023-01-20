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
	"fmt"
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

// FindID finds dataset attributes
func FindID(stmt string, args ...interface{}) (int64, error) {
	var rid int64
	err := FilesDB.QueryRow(stmt, args...).Scan(&rid)
	if err == nil {
		return rid, nil
	}
	return -1, errors.New("Unable to find id")
}

// InsertFiles insert given files into FilesDB
func InsertFiles(did int64, dataset, path string) error {
	// look-up files for given path
	files := FindFiles(path)

	// dataset is a /cycle/beamline/BTR/sample
	arr := strings.Split(dataset, "/")
	if len(arr) != 5 {
		return errors.New(fmt.Sprintf("ERROR: unable to parse given dataset %s", dataset))
	}
	cycle := arr[1]
	beamline := arr[2]
	btr := arr[3]
	sample := arr[4]
	log.Printf("InsertFiles: parse dataset=%s to cycle=%s beamline=%s btr=%s sample=%s", dataset, cycle, beamline, btr, sample)

	// check if we have already our dataset in DB
	dstmt := "SELECT dataset_id FROM datasets JOIN cules ON datasets.cycle_id=cycles.cycle_id JOIN btrs ON datasets.btr_id=btrs.btr_id JOIN beamlines ON datasets.beamline_id=beamlines.beamline_id JOIN samples ON dataset.sample_id=samples.sample_id WHERE beamlines.name=? and btrs.name=? and cycles.name=? and samaples.name=?"
	datasetID, e := FindID(dstmt, cycle, beamline, btr, sample)
	if e == nil && datasetID == did {
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

	var res []Record
	var stmt string
	// insert main attributes
	stmt = "INSERT INTO cycles (name) VALUES (?)"
	_, err = tx.Exec(stmt, cycle)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, cycle, err)
		return tx.Rollback()
	}
	stmt = "INSERT INTO beamlines (name) VALUES (?)"
	_, err = tx.Exec(stmt, beamline)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, beamline, err)
		return tx.Rollback()
	}
	stmt = "INSERT INTO btrs (name) VALUES (?)"
	_, err = tx.Exec(stmt, btr)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, btr, err)
		return tx.Rollback()
	}
	stmt = "INSERT INTO samples (name) VALUES (?)"
	_, err = tx.Exec(stmt, sample)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, sample, err)
		return tx.Rollback()
	}

	// select main attributes ids
	var rec Record

	stmt = "SELECT cycle_id FROM cycles WHERE name=?"
	res, err = execute(tx, stmt, cycle)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, cycle, err)
		return tx.Rollback()
	}
	rec = res[0]
	cycleId := rec["cycle_id"].(int64)

	stmt = "SELECT beamline_id FROM beamlines WHERE name=?"
	res, err = execute(tx, stmt, beamline)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, beamline, err)
		return tx.Rollback()
	}
	rec = res[0]
	beamlineId := rec["beamline_id"].(int64)

	stmt = "SELECT btr_id FROM btrs WHERE name=?"
	res, err = execute(tx, stmt, btr)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, btr, err)
		return tx.Rollback()
	}
	rec = res[0]
	btrId := rec["btr_id"].(int64)

	stmt = "SELECT sample_id FROM samples WHERE name=?"
	res, err = execute(tx, stmt, sample)
	if err != nil {
		log.Printf("ERROR: unable to execute %s with %v, error=%v", stmt, sample, err)
		return tx.Rollback()
	}
	rec = res[0]
	sampleId := rec["sample_id"].(int64)

	// insert data into datasets table
	tstamp := time.Now()
	stmt = "INSERT INTO datasets (dataset_id,cycle_id,beamline_id,btr_id,sample_id,tstamp) VALUES (?, ?, ?, ?, ?, ?)"
	_, err = tx.Exec(stmt, did, cycleId, beamlineId, btrId, sampleId, tstamp)
	if err != nil {
		log.Printf("ERROR: unable to execute %s, datasetId=%v, cycleId=%v, beamlineId=%v, btrId=%v, sampleId=%v, tstamp=%v, error=%v", stmt, did, cycleId, beamlineId, btrId, sampleId, tstamp, err)
		return tx.Rollback()
	}

	// insert files info
	for _, name := range files {
		stmt = "INSERT INTO files (dataset_id,name) VALUES (?,?)"
		_, err = tx.Exec(stmt, did, name)
		if err != nil {
			log.Printf("ERROR: unable to execute %s with did=%v name=%s error=%v", stmt, did, name, err)
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
func getFiles(did int64) ([]string, error) {
	var files []string
	// proceed with transaction operation
	tx, err := FilesDB.Begin()
	if err != nil {
		log.Printf("ERROR: DB error %v\n", err)
		return files, err
	}
	defer tx.Rollback()
	// look-up files info
	stmt := "SELECT name FROM files WHERE dataset_id=?"
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
