package template

import (
	"testing"
)

func TestNewMap (t *testing.T) {
	m := NewMap()

	if m.data == nil {
		t.Errorf("expected data to not be nil")
	}
}

func TestMapPut (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)
}

func TestMapPutGet (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	rt := m.Get("a")
	if rt.(string) != "ABC" {
		t.Errorf("expected value is not be correct. Expected: 'ABC' Got: %s" , rt )
	}

	rt = m.Get("1")
	if rt.(int) != 123 {
		t.Errorf("expected value is not be correct. Expected: 123 Got: %s", rt )
	}

	if m.Size() != 2 {
		t.Errorf("expected size is not be correct. Expected: 2 Got: %i", m.Size() )
	}
}

func TestMapPutClear (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	if m.Size() != 2 {
		t.Errorf("expected size is not be correct. Expected: 2 Got: %i", m.Size() )
	}

	m.Clear()

	if m.Size() != 0 {
		t.Errorf("expected size is not be correct. Expected: 0 Got: %i", m.Size() )
	}
}

func TestMapPutClearGet (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	if m.Size() != 2 {
		t.Errorf("expected size is not be correct. Expected: 2 Got: %i", m.Size() )
	}

	m.Clear()

	if m.Size() != 0 {
		t.Errorf("expected size is not be correct. Expected: 0 Got: %i", m.Size() )
	}

	rt := m.Get("a")
	if rt != nil {
		t.Errorf("expected value is not be correct. Expected: nil Got: %s", rt)
	}
	rt = m.Get("1")
	if rt != nil {
		t.Errorf("expected value is not be correct. Expected: nil Got: %s", rt)
	}
}

func TestMapPutRemoveGet (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	if m.Size() != 2 {
		t.Errorf("expected size is not be correct. Expected: 2 Got: %i", m.Size() )
	}

	m.Remove("a")
	m.Remove("1")

	if m.Size() != 0 {
		t.Errorf("expected size is not be correct. Expected: 0 Got: %i", m.Size() )
	}

	rt := m.Get("a")
	if rt != nil {
		t.Errorf("expected value is not be correct. Expected: nil Got: %s", rt)
	}

	rt = m.Get("1")
	if rt != nil {
		t.Errorf("expected value is not be correct. Expected: nil Got: %s", rt )
	}
}

func TestMapPutClearPutGet (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	if m.Size() != 2 {
		t.Errorf("expected size is not be correct. Expected: 2 Got: %i", m.Size() )
	}

	m.Clear()

	if m.Size() != 0 {
		t.Errorf("expected size is not be correct. Expected: 2 Got: %i", m.Size() )
	}

	m.Put("a", "XYZ")
	m.Put("1", 789)

	if m.Size() != 2 {
		t.Errorf("expected size is not be correct. Expected: 2 Got: %i", m.Size() )
	}

	rt := m.Get("a")
	if rt.(string) != "XYZ" {
		t.Errorf("expected value is not be correct. Expected: 'XYZ' Got: %s", rt)
	}
	rt = m.Get("1")
	if rt.(int) != 789 {
		t.Errorf("expected value is not be correct. Expected: nil Got: %s", rt)
	}
}

func TestMapPutPutGet (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("a", "MNO")
	m.Put("a", "XYZ")
	m.Put("1", 123)
	m.Put("1", 456)
	m.Put("1", 789)

	if m.Size() != 2 {
		t.Errorf("expected size is not be correct. Expected: 2 Got: %i", m.Size() )
	}

	rt := m.Get("a")
	if rt.(string) != "XYZ" {
		t.Errorf("expected value is not be correct. Expected: 'XYZ' Got: %s", rt )
	}

	rt = m.Get("1")
	if rt.(int) != 789 {
		t.Errorf("expected value is not be correct. Expected: 789 Got: %s ", rt )
	}
}

func TestMapEmptyGet (t *testing.T) {
	m := NewMap()

	if m.Size() != 0 {
		t.Errorf("expected size is not be correct. Expected: 0 Got: %i", m.Size() )
	}

	rt := m.Get("a")
	if rt != nil {
		t.Errorf("expected value is not be correct. Expected: nil Got: %s", rt )
	}

	rt = m.Get("1")
	if rt != nil {
		t.Errorf("expected value is not be correct. Expected: nil Got: %s", rt )
	}
}

// *****************

func TestMapPutContainsKey (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	rt := m.ContainsKey("a")
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s" , rt )
	}

	rt = m.ContainsKey("1")
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s", rt )
	}

}

func TestMapPutClearContainsKey (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	m.Clear()

	rt := m.ContainsKey("a")
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt)
	}
	rt = m.ContainsKey("1")
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt)
	}
}

func TestMapPutRemoveContainsKey (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	m.Remove("a")
	m.Remove("1")

	rt := m.ContainsKey("a")
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt)
	}

	rt = m.ContainsKey("1")
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt )
	}
}

func TestMapPutClearPutContainsKey (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	m.Clear()

	m.Put("a", "XYZ")
	m.Put("1", 789)

	rt := m.ContainsKey("a")
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s", rt)
	}
	rt = m.ContainsKey("1")
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s", rt)
	}
}

func TestMapPutPutContainsKey (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("a", "MNO")
	m.Put("a", "XYZ")
	m.Put("1", 123)
	m.Put("1", 456)
	m.Put("1", 789)

	rt := m.ContainsKey("a")
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s", rt )
	}

	rt = m.ContainsKey("1")
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s ", rt )
	}
}

func TestMapEmptyContainsKey (t *testing.T) {
	m := NewMap()

	rt := m.ContainsKey("a")
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt )
	}

	rt = m.ContainsKey("1")
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt )
	}
}

// *****************

func TestMapPutContainsValue (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	rt := m.ContainsValue("ABC")
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s" , rt )
	}

	rt = m.ContainsValue(123)
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s", rt )
	}

}

func TestMapPutClearContainsValue (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	m.Clear()

	rt := m.ContainsValue("ABC")
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt)
	}
	rt = m.ContainsValue(123)
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt)
	}
}

func TestMapPutRemoveContainsValue (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	m.Remove("a")
	m.Remove("1")

	rt := m.ContainsValue("ABC")
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt)
	}

	rt = m.ContainsValue(123)
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt )
	}
}

func TestMapPutClearPutContainsValue (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("1", 123)

	m.Clear()

	m.Put("a", "XYZ")
	m.Put("1", 789)

	rt := m.ContainsValue("XYZ")
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s", rt)
	}
	rt = m.ContainsValue(789)
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s", rt)
	}
}

func TestMapPutPutContainsValue (t *testing.T) {
	m := NewMap()
	m.Put("a", "ABC")
	m.Put("a", "MNO")
	m.Put("a", "XYZ")
	m.Put("1", 123)
	m.Put("1", 456)
	m.Put("1", 789)

	rt := m.ContainsValue("XYZ")
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s", rt )
	}

	rt = m.ContainsValue(789)
	if rt != true {
		t.Errorf("expected value is not be correct. Expected: true Got: %s ", rt )
	}
}

func TestMapEmptyContainsValue (t *testing.T) {
	m := NewMap()

	rt := m.ContainsValue("ABC")
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt )
	}

	rt = m.ContainsValue(123)
	if rt != false {
		t.Errorf("expected value is not be correct. Expected: false Got: %s", rt )
	}
}

func TestMapEquals (t *testing.T) {
	m := NewMap()
	m2 := NewMap()

	m.Put("a", "abc")
	m2.Put("a", "abc")
	m.Put("x", "xyz")
	m2.Put("x", "xyz")
	m.Put("1", 123)
	m2.Put("1", 123)

	if ! m.Equals(m2) {
		t.Errorf("expected value is not equal. Got: %s | %s", m, m2 )
	}

	if ! m2.Equals(m) {
		t.Errorf("expected value is not equal. Got: %s | %s", m, m2 )
	}
}

func TestMapNotEquals (t *testing.T) {
	m := NewMap()
	m2 := NewMap()

	m.Put("a", "abc")
	m2.Put("a", "abc")
	m.Put("x", "xyz")
	m2.Put("x", "xyz")
	m.Put("1", 123)

	if m.Equals(m2) {
		t.Errorf("expected value is equal. Got: %s | %s", m, m2 )
	}

	if m2.Equals(m) {
		t.Errorf("expected value is equal. Got: %s | %s", m, m2 )
	}
}

func TestMapEmptyEquals (t *testing.T) {
	m := NewMap()
	m2 := NewMap()

	if ! m.Equals(m2) {
		t.Errorf("expected value is equal. Got: %s | %s", m, m2 )
	}

	if ! m2.Equals(m) {
		t.Errorf("expected value is equal. Got: %s | %s", m, m2 )
	}
}

func TestMapEmptyNotEquals (t *testing.T) {
	m := NewMap()
	m2 := NewMap()
	m2.Put("a", "abc")

	if m.Equals(m2) {
		t.Errorf("expected value is not equal. Got: %s | %s", m, m2 )
	}

	if m2.Equals(m) {
		t.Errorf("expected value is not equal. Got: %s | %s", m, m2 )
	}
}
