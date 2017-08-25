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
	"math"
	"testing"
	"time"

	p "github.com/SAP/go-hdb/internal/protocol"
)

type testCustomInt int

func assertEqualInt(t *testing.T, dt p.DataType, v interface{}, r int64) {
	c := columnConverter(dt)
	cv, err := c.ConvertValue(v)
	if err != nil {
		t.Fatal(err)
	}
	if cv.(int64) != r {
		t.Fatalf("assert equal int failed %v - %d expected", cv, r)
	}
}

func assertEqualIntOutOfRangeError(t *testing.T, dt p.DataType, v interface{}) {
	c := columnConverter(dt)
	_, err := c.ConvertValue(v)
	if err != ErrIntegerOutOfRange {
		t.Fatalf("assert equal out of range error failed %s %v", dt, v)
	}
}

func TestConvertInteger(t *testing.T) {

	// integer data types
	assertEqualInt(t, p.DtTinyint, 42, 42)
	assertEqualInt(t, p.DtSmallint, 42, 42)
	assertEqualInt(t, p.DtInt, 42, 42)
	assertEqualInt(t, p.DtBigint, 42, 42)

	// custom integer data type
	assertEqualInt(t, p.DtInt, testCustomInt(42), 42)

	// integer reference
	i := 42
	assertEqualInt(t, p.DtBigint, &i, 42)

	// min max values
	assertEqualIntOutOfRangeError(t, p.DtTinyint, minTinyint-1)
	assertEqualIntOutOfRangeError(t, p.DtTinyint, maxTinyint+1)
	assertEqualIntOutOfRangeError(t, p.DtSmallint, minSmallint-1)
	assertEqualIntOutOfRangeError(t, p.DtSmallint, maxSmallint+1)
	assertEqualIntOutOfRangeError(t, p.DtInt, minInteger-1)
	assertEqualIntOutOfRangeError(t, p.DtInt, maxInteger+1)

}

type testCustomFloat float32

func assertEqualFloat(t *testing.T, dt p.DataType, v interface{}, r float64) {
	c := columnConverter(dt)
	cv, err := c.ConvertValue(v)
	if err != nil {
		t.Fatal(err)
	}
	if cv.(float64) != r {
		t.Fatalf("assert equal float failed %v - %f expected", cv, r)
	}
}

func assertEqualFloatOutOfRangeError(t *testing.T, dt p.DataType, v interface{}) {
	c := columnConverter(dt)
	_, err := c.ConvertValue(v)
	if err != ErrFloatOutOfRange {
		t.Fatalf("assert equal out of range error failed %s %v", dt, v)
	}
}

func TestConvertFloat(t *testing.T) {

	realValue := float32(42.42)
	doubleValue := float64(42.42)

	// float data types
	assertEqualFloat(t, p.DtReal, realValue, float64(realValue))
	assertEqualFloat(t, p.DtDouble, doubleValue, doubleValue)

	// custom float data type
	assertEqualFloat(t, p.DtReal, testCustomFloat(realValue), float64(realValue))

	// float reference
	assertEqualFloat(t, p.DtReal, &realValue, float64(realValue))

	// min max values
	assertEqualFloatOutOfRangeError(t, p.DtReal, math.Nextafter(maxReal, maxDouble))
	assertEqualFloatOutOfRangeError(t, p.DtReal, math.Nextafter(maxReal, maxDouble)*-1)

}

type testCustomTime time.Time

func assertEqualTime(t *testing.T, v interface{}, r time.Time) {
	c := columnConverter(p.DtTime)
	cv, err := c.ConvertValue(v)
	if err != nil {
		t.Fatal(err)
	}
	if !cv.(time.Time).Equal(r) {
		t.Fatalf("assert equal time failed %v - %v expected", cv, r)
	}
}

func TestConvertTime(t *testing.T) {

	timeValue := time.Now()

	// time data type
	assertEqualTime(t, timeValue, timeValue)

	// custom time data type
	assertEqualTime(t, testCustomTime(timeValue), timeValue)

	// time reference
	assertEqualTime(t, &timeValue, timeValue)

}

type testCustomString string

func assertEqualString(t *testing.T, dt p.DataType, v interface{}, r string) {
	c := columnConverter(dt)
	cv, err := c.ConvertValue(v)
	if err != nil {
		t.Fatal(err)
	}
	if cv.(string) != r {
		t.Fatalf("assert equal string failed %v - %s expected", cv, r)
	}
}

func TestConvertString(t *testing.T) {

	stringValue := "Hello World"

	// string data types
	assertEqualString(t, p.DtString, stringValue, stringValue)

	// custom string data type
	assertEqualString(t, p.DtString, testCustomString(stringValue), stringValue)

	// string reference
	assertEqualString(t, p.DtString, &stringValue, stringValue)

}

type testCustomBytes []byte

func assertEqualBytes(t *testing.T, dt p.DataType, v interface{}, r []byte) {
	c := columnConverter(dt)
	cv, err := c.ConvertValue(v)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(cv.([]byte), r) != 0 {
		t.Fatalf("assert equal bytes failed %v - %v expected", cv, r)
	}
}

func TestConvertBytes(t *testing.T) {

	bytesValue := []byte("Hello World")

	// bytes data types
	assertEqualBytes(t, p.DtString, bytesValue, bytesValue)
	assertEqualBytes(t, p.DtBytes, bytesValue, bytesValue)

	// custom bytes data type
	assertEqualBytes(t, p.DtString, testCustomBytes(bytesValue), bytesValue)
	assertEqualBytes(t, p.DtBytes, testCustomBytes(bytesValue), bytesValue)

	// bytes reference
	assertEqualBytes(t, p.DtString, &bytesValue, bytesValue)
	assertEqualBytes(t, p.DtBytes, &bytesValue, bytesValue)

}
