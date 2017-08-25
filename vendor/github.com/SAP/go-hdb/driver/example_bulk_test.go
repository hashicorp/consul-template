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
	"database/sql"
	"log"

	"github.com/SAP/go-hdb/driver"
)

// ExampleBulkInsert inserts 1000 integer values into database table test.
// Precondition: the test database table with one field of type integer must exist.
// The insert SQL command is "bulk insert" instead of "insert".
// After the insertion of the values a final stmt.Exec() without parameters must be executed.
func Example_bulkInsert() {
	db, err := sql.Open(driver.DriverName, "hdb://user:password@host:port")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("bulk insert into test values (?)") // Prepare bulk query.
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for i := 0; i < 1000; i++ {
		if _, err := stmt.Exec(i); err != nil {
			log.Fatal(err)
		}
	}
	// Call final stmt.Exec().
	if _, err := stmt.Exec(); err != nil {
		log.Fatal(err)
	}
}
