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

package driver_test

import (
	"bytes"
	"database/sql"
	"log"
	"os"

	"github.com/SAP/go-hdb/driver"
)

// ExampleLobRead reads data from a largs data object database field into a bytes.Buffer.
// Precondition: the test database table with one field of type BLOB, CLOB or NCLOB must exist.
// For illustrative purposes we assume, that the database table has exactly one record, so
// that we can use db.QueryRow.
func ExampleLob_read() {
	b := new(bytes.Buffer)

	db, err := sql.Open("hdb", "hdb://user:password@host:port")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	lob := new(driver.Lob)
	lob.SetWriter(b) // SetWriter sets the io.Writer object, to which the database content of the lob field is written.

	if err := db.QueryRow("select * from test").Scan(lob); err != nil {
		log.Fatal(err)
	}
}

// ExampleLobWrite inserts data read from a file into a database large object field.
// Precondition: the test database table with one field of type BLOB, CLOB or NCLOB and the
// test.txt file in the working directory must exist.
// Lob fields cannot be written in hdb auto commit mode - therefore the insert has to be
// executed within a transaction.
func ExampleLob_write() {
	file, err := os.Open("test.txt") // Open file.
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	db, err := sql.Open("hdb", "hdb://user:password@host:port")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin() // Start Transaction to avoid database error: SQL Error 596 - LOB streaming is not permitted in auto-commit mode.
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("insert into test values(?)")
	if err != nil {
		log.Fatal(err)
	}

	lob := new(driver.Lob)
	lob.SetReader(file) // SetReader sets the io.Reader object, which content is written to the database lob field.

	if _, err := stmt.Exec(lob); err != nil {
		log.Fatal(err)
	}

	stmt.Close()

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}
