package mssql

import (
	"reflect"
	"testing"
	"time"
)

func TestMakeGoLangScanType(t *testing.T) {
	if (reflect.TypeOf(int64(0)) != makeGoLangScanType(typeInfo{TypeId: typeInt8})) {
		t.Errorf("invalid type returned for typeDateTime")
	}
	if (reflect.TypeOf(float64(0)) != makeGoLangScanType(typeInfo{TypeId: typeFlt4})) {
		t.Errorf("invalid type returned for typeDateTime")
	}
	if (reflect.TypeOf(float64(0)) != makeGoLangScanType(typeInfo{TypeId: typeFlt8})) {
		t.Errorf("invalid type returned for typeDateTime")
	}
	if (reflect.TypeOf("") != makeGoLangScanType(typeInfo{TypeId: typeVarChar})) {
		t.Errorf("invalid type returned for typeDateTime")
	}
	if (reflect.TypeOf(time.Time{}) != makeGoLangScanType(typeInfo{TypeId: typeDateTime})) {
		t.Errorf("invalid type returned for typeDateTime")
	}
	if (reflect.TypeOf(time.Time{}) != makeGoLangScanType(typeInfo{TypeId: typeDateTim4})) {
		t.Errorf("invalid type returned for typeDateTim4")
	}
	if (reflect.TypeOf(int64(0)) != makeGoLangScanType(typeInfo{TypeId: typeInt1})) {
		t.Errorf("invalid type returned for typeInt1")
	}
	if (reflect.TypeOf(int64(0)) != makeGoLangScanType(typeInfo{TypeId: typeInt2})) {
		t.Errorf("invalid type returned for typeInt2")
	}
	if (reflect.TypeOf(int64(0)) != makeGoLangScanType(typeInfo{TypeId: typeInt4})) {
		t.Errorf("invalid type returned for typeInt4")
	}
	if (reflect.TypeOf(int64(0)) != makeGoLangScanType(typeInfo{TypeId: typeIntN, Size: 4})) {
		t.Errorf("invalid type returned for typeIntN")
	}
}
