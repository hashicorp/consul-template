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
	"testing"
)

func TestCheckBulkInsert(t *testing.T) {

	var data = []struct {
		bulkSql    string
		sql        string
		bulkInsert bool
	}{
		{"bulk insert", "insert", true},
		{"   bulk   insert  ", "insert  ", true},
		{"BuLk iNsErT", "iNsErT", true},
		{"   bUlK   InSeRt  ", "InSeRt  ", true},
		{"  bulkinsert  ", "  bulkinsert  ", false},
		{"bulk", "bulk", false},
		{"insert", "insert", false},
	}

	for i, d := range data {
		sql, bulkInsert := checkBulkInsert(d.bulkSql)
		if sql != d.sql {
			t.Fatalf("test %d failed: bulk insert flag %t - %t expected", i, bulkInsert, d.bulkInsert)
		}
		if sql != d.sql {
			t.Fatalf("test %d failed: sql %s - %s expected", i, sql, d.sql)
		}
	}
}

func TestInsertByQuery(t *testing.T) {

	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	table := RandomIdentifier("insertByQuery_")
	if _, err := db.Exec(fmt.Sprintf("create table %s.%s (i integer)", TestSchema, table)); err != nil {
		t.Fatal(err)
	}

	// insert value via Query
	if err := db.QueryRow(fmt.Sprintf("insert into %s.%s values (?)", TestSchema, table), 42).Scan(); err != sql.ErrNoRows {
		t.Fatal(err)
	}

	// check value
	var i int
	if err := db.QueryRow(fmt.Sprintf("select * from %s.%s", TestSchema, table)).Scan(&i); err != nil {
		t.Fatal(err)
	}
	if i != 42 {
		t.Fatalf("value %d - expected %d", i, 42)
	}
}

func TestHDBWarning(t *testing.T) {
	// procedure gives warning:
	// 	SQL HdbWarning 1347 - Not recommended feature: DDL statement is used in Dynamic SQL (current dynamic_sql_ddl_error_level = 1)
	const procOut = `create procedure %[1]s.%[2]s ()
language SQLSCRIPT as
begin
	exec 'create table %[3]s(id int)';
	exec 'drop table %[3]s';
end
`

	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	procedure := RandomIdentifier("proc_")
	tableName := RandomIdentifier("table_")

	if _, err := db.Exec(fmt.Sprintf(procOut, TestSchema, procedure, tableName)); err != nil { // Create stored procedure.
		t.Fatal(err)
	}

	if _, err := db.Exec(fmt.Sprintf("call %s.%s", TestSchema, procedure)); err != nil {
		t.Fatal(err)
	}
}
