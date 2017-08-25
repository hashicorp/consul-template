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
	"math/big"
	"reflect"
	"testing"
	"time"
)

func TestTinyint(t *testing.T) {
	testDatatype(t, "tinyint", 0, true,
		uint8(minTinyint),
		uint8(maxTinyint),
		sql.NullInt64{Valid: false, Int64: minTinyint},
		sql.NullInt64{Valid: true, Int64: maxTinyint},
	)
}

func TestSmallint(t *testing.T) {
	testDatatype(t, "smallint", 0, true,
		int16(minSmallint),
		int16(maxSmallint),
		sql.NullInt64{Valid: false, Int64: minSmallint},
		sql.NullInt64{Valid: true, Int64: maxSmallint},
	)
}

func TestInteger(t *testing.T) {
	testDatatype(t, "integer", 0, true,
		int32(minInteger),
		int32(maxInteger),
		sql.NullInt64{Valid: false, Int64: minInteger},
		sql.NullInt64{Valid: true, Int64: maxInteger},
	)
}

func TestBigint(t *testing.T) {
	testDatatype(t, "bigint", 0, true,
		int64(minBigint),
		int64(maxBigint),
		sql.NullInt64{Valid: false, Int64: minBigint},
		sql.NullInt64{Valid: true, Int64: maxBigint},
	)
}

func TestReal(t *testing.T) {
	testDatatype(t, "real", 0, true,
		float32(-maxReal),
		float32(maxReal),
		sql.NullFloat64{Valid: false, Float64: -maxReal},
		sql.NullFloat64{Valid: true, Float64: maxReal},
	)
}

func TestDouble(t *testing.T) {
	testDatatype(t, "double", 0, true,
		float64(-maxDouble),
		float64(maxDouble),
		sql.NullFloat64{Valid: false, Float64: -maxDouble},
		sql.NullFloat64{Valid: true, Float64: maxDouble},
	)
}

var testStringDataAscii = []interface{}{
	"Hello HDB",
	"aaaaaaaaaa",
	sql.NullString{Valid: false, String: "Hello HDB"},
	sql.NullString{Valid: true, String: "Hello HDB"},
}

var testStringData = []interface{}{
	"Hello HDB",
	// varchar: UTF-8 4 bytes per char -> size 40 bytes
	// nvarchar: CESU-8 6 bytes per char -> hdb counts 2 chars per 6 byte encoding -> size 20 bytes
	"𝄞𝄞𝄞𝄞𝄞𝄞𝄞𝄞𝄞𝄞",
	"𝄞𝄞aa",
	"€€",
	"𝄞𝄞€€",
	"𝄞𝄞𝄞€€",
	"aaaaaaaaaa",
	sql.NullString{Valid: false, String: "Hello HDB"},
	sql.NullString{Valid: true, String: "Hello HDB"},
}

/*
using unicode (CESU-8) data for char HDB
- successful insert into table
- but query table returns
  SQL HdbError 7 - feature not supported: invalid character encoding: ...
--> use ASCII test data only
surprisingly: varchar works with unicode characters
*/
func TestChar(t *testing.T) {
	testDatatype(t, "char", 40, true, testStringDataAscii...)
}

func TestVarchar(t *testing.T) {
	testDatatype(t, "varchar", 40, false, testStringData...)
}

func TestNChar(t *testing.T) {
	testDatatype(t, "nchar", 20, true, testStringData...)
}

func TestNVarchar(t *testing.T) {
	testDatatype(t, "nvarchar", 20, false, testStringData...)
}

var testBinaryData = []interface{}{
	[]byte("Hello HDB"),
	[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
	[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0xff},
	NullBytes{Valid: false, Bytes: []byte("Hello HDB")},
	NullBytes{Valid: true, Bytes: []byte("Hello HDB")},
}

func TestBinary(t *testing.T) {
	testDatatype(t, "binary", 20, true, testBinaryData...)
}

func TestVarbinary(t *testing.T) {
	testDatatype(t, "varbinary", 20, false, testBinaryData...)
}

var testTimeData = []interface{}{
	time.Now(),
	NullTime{Valid: false, Time: time.Now()},
	NullTime{Valid: true, Time: time.Now()},
}

func TestDate(t *testing.T) {
	testDatatype(t, "date", 0, true, testTimeData...)
}

func TestTime(t *testing.T) {
	testDatatype(t, "time", 0, true, testTimeData...)
}

func TestTimestamp(t *testing.T) {
	testDatatype(t, "timestamp", 0, true, testTimeData...)
}

func TestSeconddate(t *testing.T) {
	testDatatype(t, "seconddate", 0, true, testTimeData...)
}

var testDecimalData = []interface{}{
	(*Decimal)(big.NewRat(0, 1)),
	(*Decimal)(big.NewRat(1, 1)),
	(*Decimal)(big.NewRat(-1, 1)),
	(*Decimal)(big.NewRat(10, 1)),
	(*Decimal)(big.NewRat(1000, 1)),
	(*Decimal)(big.NewRat(1, 10)),
	(*Decimal)(big.NewRat(-1, 10)),
	(*Decimal)(big.NewRat(1, 1000)),
	(*Decimal)(new(big.Rat).SetInt(maxDecimal)),
	NullDecimal{Valid: false, Decimal: (*Decimal)(big.NewRat(1, 1))},
	NullDecimal{Valid: true, Decimal: (*Decimal)(big.NewRat(1, 1))},
}

func TestDecimal(t *testing.T) {
	testDatatype(t, "decimal", 0, true, testDecimalData...)
}

//
func testDatatype(t *testing.T, dataType string, dataSize int, fixedSize bool, testData ...interface{}) {
	db, err := sql.Open(DriverName, TestDsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	table := RandomIdentifier(fmt.Sprintf("%s_", dataType))
	if dataSize == 0 {
		if _, err := db.Exec(fmt.Sprintf("create table %s.%s (i integer, x %s)", TestSchema, table, dataType)); err != nil {
			t.Fatal(err)
		}
	} else {
		if _, err := db.Exec(fmt.Sprintf("create table %s.%s (i integer, x %s(%d))", TestSchema, table, dataType, dataSize)); err != nil {
			t.Fatal(err)
		}

	}

	stmt, err := db.Prepare(fmt.Sprintf("insert into %s.%s values(?, ?)", TestSchema, table))
	if err != nil {
		t.Fatal(err)
	}

	for i, in := range testData {
		if _, err := stmt.Exec(i, in); err != nil {
			t.Fatal(err)
		}
	}

	size := len(testData)
	var i int

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

		in := testData[i]
		out := reflect.New(reflect.TypeOf(in)).Interface()

		switch out := out.(type) {
		case *NullDecimal:
			out.Decimal = (*Decimal)(new(big.Rat))
		}

		if err := rows.Scan(&i, out); err != nil {
			log.Fatal(err)
		}

		switch out := out.(type) {
		default:
			t.Fatalf("%d unknown type %T", i, out)
		case *uint8:
			if *out != in.(uint8) {
				t.Fatalf("%d value %v - expected %v", i, *out, in)
			}
		case *int16:
			if *out != in.(int16) {
				t.Fatalf("%d value %v - expected %v", i, *out, in)
			}
		case *int32:
			if *out != in.(int32) {
				t.Fatalf("%d value %v - expected %v", i, *out, in)
			}
		case *int64:
			if *out != in.(int64) {
				t.Fatalf("%d value %v - expected %v", i, *out, in)
			}
		case *float32:
			if *out != in.(float32) {
				t.Fatalf("%d value %v - expected %v", i, *out, in)
			}
		case *float64:
			if *out != in.(float64) {
				t.Fatalf("%d value %v - expected %v", i, *out, in)
			}
		case *string:
			if fixedSize {
				if !compareStringFixSize(in.(string), *out) {
					t.Fatalf("%d value %v - expected %v", i, *out, in)
				}
			} else {
				if *out != in.(string) {
					t.Fatalf("%d value %v - expected %v", i, *out, in)
				}
			}
		case *[]byte:
			if fixedSize {
				if !compareBytesFixSize(in.([]byte), *out) {
					t.Fatalf("%d value %v - expected %v", i, *out, in)
				}
			} else {
				if bytes.Compare(*out, in.([]byte)) != 0 {
					t.Fatalf("%d value %v - expected %v", i, *out, in)
				}
			}
		case **Decimal:
			if ((*big.Rat)(*out)).Cmp((*big.Rat)(in.(*Decimal))) != 0 {
				t.Fatalf("%d value %s - expected %s", i, (*big.Rat)(*out), (*big.Rat)(in.(*Decimal)))
			}
		case *time.Time:
			switch dataType {
			default:
				t.Fatalf("unknown data type %s", dataType)
			case "date":
				if !equalDate(*out, in.(time.Time)) {
					t.Fatalf("%d value %v - expected %v", i, *out, in)
				}
			case "time":
				if !equalTime(*out, in.(time.Time)) {
					t.Fatalf("%d value %v - expected %v", i, *out, in)
				}
			case "timestamp":
				if !equalTimestamp(*out, in.(time.Time)) {
					t.Fatalf("%d value %v - expected %v", i, *out, in)
				}
			case "seconddate":
				if !equalDateTime(*out, in.(time.Time)) {
					t.Fatalf("%d value %v - expected %v", i, *out, in)
				}
			}
		case *sql.NullInt64:
			in := in.(sql.NullInt64)
			if in.Valid != out.Valid {
				t.Fatalf("%d value %v - expected %v", i, out, in)
			}
			if in.Valid && in.Int64 != out.Int64 {
				t.Fatalf("%d value %v - expected %v", i, out, in)
			}
		case *sql.NullFloat64:
			in := in.(sql.NullFloat64)
			if in.Valid != out.Valid {
				t.Fatalf("%d value %v - expected %v", i, out, in)
			}
			if in.Valid && in.Float64 != out.Float64 {
				t.Fatalf("%d value %v - expected %v", i, out, in)
			}
		case *sql.NullString:
			in := in.(sql.NullString)
			if in.Valid != out.Valid {
				t.Fatalf("%d value %v - expected %v", i, out, in)
			}
			if in.Valid {
				if fixedSize {
					if !compareStringFixSize(in.String, out.String) {
						t.Fatalf("%d value %v - expected %v", i, *out, in)
					}
				} else {
					if in.String != out.String {
						t.Fatalf("%d value %v - expected %v", i, out, in)
					}
				}
			}
		case *NullBytes:
			in := in.(NullBytes)
			if in.Valid != out.Valid {
				t.Fatalf("%d value %v - expected %v", i, out, in)
			}
			if in.Valid {
				if fixedSize {
					if !compareBytesFixSize(in.Bytes, out.Bytes) {
						t.Fatalf("%d value %v - expected %v", i, *out, in)
					}
				} else {
					if bytes.Compare(in.Bytes, out.Bytes) != 0 {
						t.Fatalf("%d value %v - expected %v", i, out, in)
					}
				}
			}
		case *NullTime:
			in := in.(NullTime)
			if in.Valid != out.Valid {
				t.Fatalf("%d value %v - expected %v", i, out, in)
			}
			if in.Valid {
				switch dataType {
				default:
					t.Fatalf("unknown data type %s", dataType)
				case "date":
					if !equalDate(out.Time, in.Time) {
						t.Fatalf("%d value %v - expected %v", i, *out, in)
					}
				case "time":
					if !equalTime(out.Time, in.Time) {
						t.Fatalf("%d value %v - expected %v", i, *out, in)
					}
				case "timestamp":
					if !equalTimestamp(out.Time, in.Time) {
						t.Fatalf("%d value %v - expected %v", i, *out, in)
					}
				case "seconddate":
					if !equalDateTime(out.Time, in.Time) {
						t.Fatalf("%d value %v - expected %v", i, *out, in)
					}
				}
			}
		case *NullDecimal:
			in := in.(NullDecimal)
			if in.Valid != out.Valid {
				t.Fatalf("%d value %v - expected %v", i, out, in)
			}
			if in.Valid {
				if ((*big.Rat)(in.Decimal)).Cmp((*big.Rat)(out.Decimal)) != 0 {
					t.Fatalf("%d value %s - expected %s", i, (*big.Rat)(in.Decimal), (*big.Rat)(in.Decimal))
				}
			}
		}
		i++
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

// helper
func compareStringFixSize(in, out string) bool {
	if in != out[:len(in)] {
		return false
	}
	for _, r := range out[len(in):] {
		if r != rune(' ') {
			return false
		}
	}
	return true
}

func compareBytesFixSize(in, out []byte) bool {
	if bytes.Compare(in, out[:len(in)]) != 0 {
		return false
	}
	for _, r := range out[len(in):] {
		if r != 0 {
			return false
		}
	}
	return true
}

func equalDate(t1, t2 time.Time) bool {
	u1 := t1.UTC()
	u2 := t2.UTC()
	return u1.Year() == u2.Year() && u1.Month() == u2.Month() && u1.Day() == u2.Day()
}

func equalTime(t1, t2 time.Time) bool {
	u1 := t1.UTC()
	u2 := t2.UTC()
	return u1.Hour() == u2.Hour() && u1.Minute() == u2.Minute() && u1.Second() == u2.Second()
}

func equalDateTime(t1, t2 time.Time) bool {
	return equalDate(t1, t2) && equalTime(t1, t2)
}

// equalMillisecond tests if the nanosecond part of two time types rounded to milliseconds are equal.
func equalMilliSecond(t1, t2 time.Time) bool {
	u1 := t1.UTC()
	u2 := t2.UTC()
	return u1.Round(time.Millisecond).Nanosecond() == u2.Round(time.Millisecond).Nanosecond()
}

func equalTimestamp(t1, t2 time.Time) bool {
	return equalDate(t1, t2) && equalTime(t1, t2) && equalMilliSecond(t1, t2)
}
