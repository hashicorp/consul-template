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
)

/*
ExampleCallSimpleOut creates a stored procedure with one output parameter and executes it.
Stored procedures with output parameters must be executed by sql.Query or sql.QueryRow.
For variables TestDsn and TestSchema see main_test.go.
*/
func Example_callSimpleOut() {
	const procOut = `create procedure %s.%s (out message nvarchar(1024))
language SQLSCRIPT as
begin
    message := 'Hello World!';
end
`

	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	procedure := RandomIdentifier("procOut_")

	if _, err := db.Exec(fmt.Sprintf(procOut, TestSchema, procedure)); err != nil { // Create stored procedure.
		log.Fatal(err)
	}

	var out string

	if err := db.QueryRow(fmt.Sprintf("call %s.%s(?)", TestSchema, procedure)).Scan(&out); err != nil {
		log.Fatal(err)
	}

	fmt.Print(out)

	// output: Hello World!
}

/*
ExampleCallTableOut creates a stored procedure with one table output parameter and executes it.
Stored procedures with table output parameters must be executed by sql.Query as sql.QueryRow will close
the query after execution and prevent querying output table values.
The scan type of a table output parameter is a string containing an opaque value to query table output values
by standard sql.Query or sql.QueryRow methods.
For variables TestDsn and TestSchema see main_test.go.
*/
func Example_callTableOut() {
	const procTable = `create procedure %[1]s.%[2]s (out t %[1]s.%[3]s)
language SQLSCRIPT as
begin
  create local temporary table #test like %[1]s.%[3]s;
  insert into #test values('Hello, 世界');
  insert into #test values('SAP HANA');
  insert into #test values('Go driver');
  t = select * from #test;
  drop table #test;
end
`

	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tableType := RandomIdentifier("TableType_")
	procedure := RandomIdentifier("ProcTable_")

	if _, err := db.Exec(fmt.Sprintf("create type %s.%s as table (x nvarchar(256))", TestSchema, tableType)); err != nil { // Create table type.
		log.Fatal(err)
	}

	if _, err := db.Exec(fmt.Sprintf(procTable, TestSchema, procedure, tableType)); err != nil { // Create stored procedure.
		log.Fatal(err)
	}

	var tableQuery string // Scan variable of table output parameter.

	// Query stored procedure.
	rows, err := db.Query(fmt.Sprintf("call %s.%s(?)", TestSchema, procedure))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		log.Fatal(rows.Err())
	}
	if err := rows.Scan(&tableQuery); err != nil {
		log.Fatal(err)
	}

	// Query stored procedure output table.
	tableRows, err := db.Query(tableQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer tableRows.Close()

	for tableRows.Next() {
		var x string

		if err := tableRows.Scan(&x); err != nil {
			log.Fatal(err)
		}

		fmt.Println(x)
	}
	if err := tableRows.Err(); err != nil {
		log.Fatal(err)
	}

	// output: Hello, 世界
	// SAP HANA
	// Go driver
}
