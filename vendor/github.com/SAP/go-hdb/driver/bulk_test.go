/*
Copyright 2014 SAP SE

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"database/sql"
	"fmt"
	"log"
	"testing"
)

const (
	bulkSamples = 10000
)

// TestBulkInsert
func TestBulkInsert(t *testing.T) {

	tmpTableName := Identifier("#tmpTable")

	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	//keep connection / hdb session for using local temporary tables
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //cleanup

	if _, err := tx.Exec(fmt.Sprintf("create local temporary table %s (i integer)", tmpTableName)); err != nil {
		t.Fatalf("create table failed: %s", err)
	}

	stmt, err := tx.Prepare(fmt.Sprintf("bulk insert into %s values (?)", tmpTableName))
	if err != nil {
		t.Fatalf("prepare bulk insert failed: %s", err)
	}
	defer stmt.Close()

	for i := 0; i < bulkSamples; i++ {
		if _, err := stmt.Exec(i); err != nil {
			t.Fatalf("insert failed: %s", err)
		}
	}
	// final flush
	if _, err := stmt.Exec(); err != nil {
		t.Fatalf("final insert (flush) failed: %s", err)
	}

	i := 0
	err = tx.QueryRow(fmt.Sprintf("select count(*) from %s", tmpTableName)).Scan(&i)
	if _, err := stmt.Exec(); err != nil {
		t.Fatalf("select count failed: %s", err)
	}

	if i != bulkSamples {
		t.Fatalf("invalid number of records %d - %d expected", i, bulkSamples)
	}

	rows, err := tx.Query(fmt.Sprintf("select * from %s order by i", tmpTableName))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	i = 0
	for rows.Next() {

		var j int

		if err := rows.Scan(&j); err != nil {
			log.Fatal(err)
		}

		if j != i {
			t.Fatalf("value %d - expected %d", j, i)
		}

		i++
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
