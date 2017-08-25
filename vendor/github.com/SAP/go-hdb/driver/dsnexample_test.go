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
	"net/url"

	"github.com/SAP/go-hdb/driver"
)

// Dsn creates data source name with the help of the net/url package.
func dsn() string {
	dsn := &url.URL{
		Scheme: driver.DriverName,
		User:   url.UserPassword("user", "password"),
		Host:   "host:port",
	}
	return dsn.String()
}

// ExampleDsn shows how to construct a DSN (data source name) as url.
func ExampleDSN() {
	db, err := sql.Open(driver.DriverName, dsn())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
}
