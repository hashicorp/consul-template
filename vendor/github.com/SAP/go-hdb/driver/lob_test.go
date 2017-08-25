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
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/SAP/go-hdb/internal/unicode"
)

type testFile struct {
	name  string
	valid bool
}

func testFiles() ([]*testFile, error) {
	var testFiles []*testFile

	filter := func(name string) bool {
		for _, ext := range []string{} {
			if filepath.Ext(name) == ext {
				return false
			}
		}
		return true
	}

	walk := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && filter(info.Name()) {
			testFiles = append(testFiles, &testFile{name: path, valid: true})
		}
		return nil
	}

	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	filepath.Walk(root, walk)
	return testFiles, nil
}

func TestBlobFile(t *testing.T) {
	files, err := testFiles()
	if err != nil {
		t.Fatal(err)
	}
	testLobFile(t, "blob", files)
}

func TestClobFile(t *testing.T) {
	files, err := testFiles()
	if err != nil {
		t.Fatal(err)
	}
	testLobFile(t, "clob", files)
}

func TestNclobFile(t *testing.T) {
	files, err := testFiles()
	if err != nil {
		t.Fatal(err)
	}
	testLobFile(t, "nclob", files)
}

func testLobFile(t *testing.T, dataType string, testFiles []*testFile) {
	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	table := RandomIdentifier(fmt.Sprintf("%s_", dataType))
	if _, err := db.Exec(fmt.Sprintf("create table %s.%s (i integer, x %s)", TestSchema, table, dataType)); err != nil {
		t.Fatal(err)
	}

	// use trancactions:
	// SQL Error 596 - LOB streaming is not permitted in auto-commit mode
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	stmt, err := tx.Prepare(fmt.Sprintf("insert into %s.%s values(?, ?)", TestSchema, table))
	if err != nil {
		t.Fatal(err)
	}

	lob := new(Lob)

	for i, testFile := range testFiles {
		file, err := os.Open(testFile.name)
		if err != nil {
			t.Fatal(err)
		}
		lob.SetReader(file)
		if _, err := stmt.Exec(i, lob); err != nil {
			if err == unicode.ErrInvalidUtf8 {
				t.Logf("filename %s - %s", testFile.name, err)
			} else {
				t.Fatalf("filename %s - %s", testFile.name, err)
			}
			testFile.valid = false
		}
		file.Close()
	}

	size := len(testFiles)
	var i int

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	if err := db.QueryRow(fmt.Sprintf("select count(*) from %s.%s", TestSchema, table)).Scan(&i); err != nil {
		t.Fatal(err)
	}

	if i != size {
		t.Fatalf("rows %d - expected %d", i, size)
	}

	rows, err := db.Query(fmt.Sprintf("select * from %s.%s order by i", TestSchema, table))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	i = 0
	for rows.Next() {

		b := new(bytes.Buffer)
		lob.SetWriter(b)

		if err := rows.Scan(&i, lob); err != nil {
			log.Fatal(err)
		}

		testFile := testFiles[i]

		if testFile.valid {
			if err := compare(testFile.name, b); err != nil {
				t.Fatalf("filename %s - %s", testFile.name, err)
			}
		}

		i++
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func compare(filename string, lob io.Reader) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	r1 := bufio.NewReader(file)
	r2 := bufio.NewReader(lob)

	for i := 0; ; i++ {
		b1, err1 := r1.ReadByte()
		b2, err2 := r2.ReadByte()
		switch {
		case err1 == io.EOF && err2 == io.EOF:
			return nil
		case err1 != nil:
			return fmt.Errorf("unexpected file EOF at %d", i)
		case err2 != nil:
			return fmt.Errorf("unexpected lob EOF at %d", i)
		}
		if b1 != b2 {
			return fmt.Errorf("diff pos %d %x - expected %x", i, b2, b1)
		}
	}
}
