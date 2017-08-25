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
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"testing"
)

func TestCallEcho(t *testing.T) {
	const procEcho = `create procedure %[1]s.%[2]s (in idata nvarchar(25), out odata nvarchar(25))
language SQLSCRIPT as
begin
    odata := idata;
end
`

	const txt = "Hello World!"

	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	procedure := RandomIdentifier("procEcho_")

	if _, err := db.Exec(fmt.Sprintf(procEcho, TestSchema, procedure)); err != nil {
		t.Fatal(err)
	}

	var out string

	if err := db.QueryRow(fmt.Sprintf("call %s.%s(?, ?)", TestSchema, procedure), txt).Scan(&out); err != nil {
		t.Fatal(err)
	}

	if out != txt {
		t.Fatalf("value %s - expected %s", out, txt)
	}

}

func TestCallBlobEcho(t *testing.T) {
	const procBlobEcho = `create procedure %[1]s.%[2]s (in idata blob, out odata blob)
language SQLSCRIPT as
begin
  odata := idata;
end
`

	const txt = "Hello World!"

	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	procedure := RandomIdentifier("procBlobEcho_")

	if _, err := db.Exec(fmt.Sprintf(procBlobEcho, TestSchema, procedure)); err != nil {
		t.Fatal(err)
	}

	inlob := new(Lob)
	inlob.SetReader(bytes.NewReader([]byte(txt)))

	b := new(bytes.Buffer)
	outlob := new(Lob)
	outlob.SetWriter(b)

	if err := db.QueryRow(fmt.Sprintf("call %s.%s(?, ?)", TestSchema, procedure), inlob).Scan(outlob); err != nil {
		t.Fatal(err)
	}

	out := b.String()

	if out != txt {
		t.Fatalf("value %s - expected %s", out, txt)
	}
}

type testTableData struct {
	i int
	x string
}

var testTableQuery1Data = []*testTableData{
	&testTableData{0, "A"},
	&testTableData{1, "B"},
	&testTableData{2, "C"},
	&testTableData{3, "D"},
	&testTableData{4, "E"},
}

var testTableQuery2Data = []*testTableData{
	&testTableData{0, "A"},
	&testTableData{1, "B"},
	&testTableData{2, "C"},
	&testTableData{3, "D"},
	&testTableData{4, "E"},
	&testTableData{5, "F"},
	&testTableData{6, "G"},
	&testTableData{7, "H"},
	&testTableData{8, "I"},
	&testTableData{9, "J"},
}

var testTableQuery3Data = []*testTableData{
	&testTableData{0, "A"},
	&testTableData{1, "B"},
	&testTableData{2, "C"},
	&testTableData{3, "D"},
	&testTableData{4, "E"},
	&testTableData{5, "F"},
	&testTableData{6, "G"},
	&testTableData{7, "H"},
	&testTableData{8, "I"},
	&testTableData{9, "J"},
	&testTableData{10, "K"},
	&testTableData{11, "L"},
	&testTableData{12, "M"},
	&testTableData{13, "N"},
	&testTableData{14, "O"},
}

func checkTableQueryData(t *testing.T, db *sql.DB, query string, data []*testTableData) {

	rows, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	j := 0
	for rows.Next() {

		var i int
		var x string

		if err := rows.Scan(&i, &x); err != nil {
			log.Fatal(err)
		}

		// log.Printf("i %d x %s", i, x)
		if i != data[j].i {
			t.Fatalf("value i %d - expected %d", i, data[j].i)
		}
		if x != data[j].x {
			t.Fatalf("value x %s - expected %s", x, data[j].x)
		}
		j++
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func TestCallTableOut(t *testing.T) {
	const procTableOut = `create procedure %[1]s.%[2]s (in i integer, out t1 %[1]s.%[3]s, out t2 %[1]s.%[3]s, out t3 %[1]s.%[3]s)
language SQLSCRIPT as
begin
  create local temporary table #test like %[1]s.%[3]s;
  insert into #test values(0, 'A');
  insert into #test values(1, 'B');
  insert into #test values(2, 'C');
  insert into #test values(3, 'D');
  insert into #test values(4, 'E');
  t1 = select * from #test;
  insert into #test values(5, 'F');
  insert into #test values(6, 'G');
  insert into #test values(7, 'H');
  insert into #test values(8, 'I');
  insert into #test values(9, 'J');
  t2 = select * from #test;
  insert into #test values(10, 'K');
  insert into #test values(11, 'L');
  insert into #test values(12, 'M');
  insert into #test values(13, 'N');
  insert into #test values(14, 'O');
  t3 = select * from #test;
  drop table #test;
end
`

	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tableType := RandomIdentifier("tt2_")
	procedure := RandomIdentifier("procTableOut_")

	if _, err := db.Exec(fmt.Sprintf("create type %s.%s as table (i integer, x varchar(10))", TestSchema, tableType)); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(fmt.Sprintf(procTableOut, TestSchema, procedure, tableType)); err != nil {
		t.Fatal(err)
	}

	var tableQuery1, tableQuery2, tableQuery3 string

	rows, err := db.Query(fmt.Sprintf("call %s.%s(?, ?, ?, ?)", TestSchema, procedure), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		log.Fatal(rows.Err())
	}
	if err := rows.Scan(&tableQuery1, &tableQuery2, &tableQuery3); err != nil {
		log.Fatal(err)
	}

	checkTableQueryData(t, db, tableQuery1, testTableQuery1Data)
	checkTableQueryData(t, db, tableQuery2, testTableQuery2Data)
	checkTableQueryData(t, db, tableQuery3, testTableQuery3Data)

}
