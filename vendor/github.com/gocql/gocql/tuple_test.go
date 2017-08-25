// +build all integration

package gocql

import (
	"reflect"
	"testing"
)

func TestTupleSimple(t *testing.T) {
	session := createSession(t)
	defer session.Close()
	if session.cfg.ProtoVersion < protoVersion3 {
		t.Skip("tuple types are only available of proto>=3")
	}

	err := createTable(session, `CREATE TABLE gocql_test.tuple_test(
		id int,
		coord frozen<tuple<int, int>>,

		primary key(id))`)
	if err != nil {
		t.Fatal(err)
	}

	err = session.Query("INSERT INTO tuple_test(id, coord) VALUES(?, (?, ?))", 1, 100, -100).Exec()
	if err != nil {
		t.Fatal(err)
	}

	var (
		id    int
		coord struct {
			x int
			y int
		}
	)

	iter := session.Query("SELECT id, coord FROM tuple_test WHERE id=?", 1)
	if err := iter.Scan(&id, &coord.x, &coord.y); err != nil {
		t.Fatal(err)
	}

	if id != 1 {
		t.Errorf("expected to get id=1 got: %v", id)
	}
	if coord.x != 100 {
		t.Errorf("expected to get coord.x=100 got: %v", coord.x)
	}
	if coord.y != -100 {
		t.Errorf("expected to get coord.y=-100 got: %v", coord.y)
	}
}

func TestTupleMapScan(t *testing.T) {
	session := createSession(t)
	defer session.Close()
	if session.cfg.ProtoVersion < protoVersion3 {
		t.Skip("tuple types are only available of proto>=3")
	}

	err := createTable(session, `CREATE TABLE gocql_test.tuple_map_scan(
		id int,
		val frozen<tuple<int, int>>,

		primary key(id))`)
	if err != nil {
		t.Fatal(err)
	}

	if err := session.Query(`INSERT INTO tuple_map_scan (id, val) VALUES (1, (1, 2));`).Exec(); err != nil {
		t.Fatal(err)
	}

	m := make(map[string]interface{})
	err = session.Query(`SELECT * FROM tuple_map_scan`).MapScan(m)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTuple_NestedCollection(t *testing.T) {
	session := createSession(t)
	defer session.Close()
	if session.cfg.ProtoVersion < protoVersion3 {
		t.Skip("tuple types are only available of proto>=3")
	}

	err := createTable(session, `CREATE TABLE gocql_test.nested_tuples(
		id int,
		val list<frozen<tuple<int, text>>>,

		primary key(id))`)
	if err != nil {
		t.Fatal(err)
	}

	type typ struct {
		A int
		B string
	}

	tests := []struct {
		name string
		val  interface{}
	}{
		{name: "slice", val: [][]interface{}{{1, "2"}, {3, "4"}}},
		{name: "array", val: [][2]interface{}{{1, "2"}, {3, "4"}}},
		{name: "struct", val: []typ{{1, "2"}, {3, "4"}}},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := session.Query(`INSERT INTO nested_tuples (id, val) VALUES (?, ?);`, i, test.val).Exec(); err != nil {
				t.Fatal(err)
			}

			rv := reflect.ValueOf(test.val)
			res := reflect.New(rv.Type()).Elem().Addr().Interface()

			err = session.Query(`SELECT val FROM nested_tuples WHERE id=?`, i).Scan(res)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
