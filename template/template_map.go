package template

import (
	"sync"
	"encoding/json"
	"bytes"
	"fmt"
	"reflect"
	"errors"
)

var (
	errBadComparisonType = errors.New("invalid type for comparison")
	errBadComparison     = errors.New("incompatible types for comparison")
	errNoComparison      = errors.New("missing argument for comparison")
)

type Map struct {
	mu sync.RWMutex

	// data is the map of everything
	data map[string]interface{}

}

func NewMap() *Map {
	return &Map{
		data: make(map[string]interface{}),
	}
}

func NewMapFromJSON(s string) *Map {
	m := NewMap()
	m.mu.Lock()

	if s == "" {
		m.data = map[string]interface{}{}
	}
	m.mu.Unlock()

	if err := json.Unmarshal([]byte(s), &m.data); err != nil {
		fmt.Printf("err, %+v", err);
	}

	return m
}


/**
Java Map Interface

void  clear()
	Removes all of the mappings from this map (optional operation).
boolean   containsKey(Object key)
	Returns true if this map contains a mapping for the specified key.
boolean   containsValue(Object value)
	Returns true if this map maps one or more keys to the specified value.
Set<Map.Entry<K,V>>   entrySet()
	Returns a Set view of the mappings contained in this map.
boolean   equals(Object o)
	Compares the specified object with this map for equality.
V   get(Object key)
	Returns the value to which the specified key is mapped, or null if this map contains no mapping for the key.
int   hashCode()
	Returns the hash code value for this map.
boolean   isEmpty()
	Returns true if this map contains no key-value mappings.
Set<K>  keySet()
	Returns a Set view of the keys contained in this map.
V   put(K key, V value)
	Associates the specified value with the specified key in this map (optional operation).
void  putAll(Map<? extends K,? extends V> m)
	Copies all of the mappings from the specified map to this map (optional operation).
V   remove(Object key)
	Removes the mapping for a key from this map if it is present (optional operation).
int   size()
	Returns the number of key-value mappings in this map.
Collection<V>   values()
	Returns a Collection view of the values contained in this map.
*/


// void  clear()
//    Removes all of the mappings from this map (optional operation).
func (m Map) Clear () {
	m.mu.Lock()
	for k := range m.data {
		delete(m.data, k)
	}
	m.mu.Unlock()

	// m.data = make(map[string]interface{}) // not working?
}

// same as Clear()
// Template requires a return, this version returns data
func (m Map) ClearReturnData () map[string]interface{}{
	m.Clear()

	return m.data
}

// same as Clear()
// Template requires a return, this version returns nothing
func (m Map) ClearReturnBlank () string {
	m.Clear()
	return ""
}

// boolean   containsKey(Object key)
//    Returns true if this map contains a mapping for the specified key.
func (m *Map) ContainsKey(k string) bool {
	m.mu.RLock()
	if _, ok := m.data[k]; ok {
		m.mu.RUnlock()
		return true
	}
	m.mu.RUnlock()
	return false
}

// boolean   containsValue(Object value)
//    Returns true if this map maps one or more keys to the specified value.
func (m *Map) ContainsValue(v interface{}) bool {
	m.mu.RLock()
	for k := range m.data {
		if v ==  m.data[k] {
			m.mu.RUnlock()
			return true
		}
	}
	m.mu.RUnlock()
	return false
}

/*

// Set<Map.Entry<K,V>>   entrySet()
//     Returns a Set view of the mappings contained in this map.
func (m *Map) EntrySet() []interface{} {
	// not yet implemented
}

*/

// boolean   equals(Object o)
//    Compares the specified object with this map for equality.
func (m *Map) Equals(m2 *Map) bool {

	m.mu.RLock()
	m2.mu.RLock()
	eq := reflect.DeepEqual(m.data, m2.data)
	m.mu.RUnlock()
	m2.mu.RUnlock()

	return eq
}



// V   get(Object key)
//    Returns the value to which the specified key is mapped, or null if this map contains no mapping for the key.
func (m *Map) Get(k string) interface{} {
	m.mu.RLock()
	value := m.data[k]
	m.mu.RUnlock()

	return value
}

// as Get
// this sometimes solves the issue in of $value is <nil> 
//     {{- $map.Put "key" (key "/path") -}}
//     {{- $value := $map.Get "key" -}}
//     {{- if eq "value_to_check" $value -}}
//       IS EQUALS
//     {{- end -}}
// 
// calling $map.GetString "key" guarantees a value of type string

func (m *Map) GetString(k string) string {
	value := m.Get(k) 

	v := reflect.ValueOf(value)
	if v.Kind() != reflect.String {
		return ""
	}

	return value.(string)
}

/*

// int   hashCode()
//    Returns the hash code value for this map.
func (m *Map) HashCode() int {
	// not yet implemented
}

*/

// boolean   isEmpty()
//    Returns true if this map contains no key-value mappings.
func (m *Map) IsEmpty() bool {
	m.mu.RLock()
	if len(m.data) > 0 {
		m.mu.RUnlock()
		return false
	}
	m.mu.RUnlock()
	return true
}

// Set<K>  keySet()
//    Returns a Set view of the keys contained in this map.
func (m *Map) KeySet() []string {
	m.mu.RLock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	m.mu.RUnlock()
	return keys
}

// V   put(K key, V value)
//    Associates the specified value with the specified key in this map (optional operation).
func (m *Map) Put(k string, v interface{}) interface{} {
	m.mu.Lock()
	m.data[k] = v
	m.mu.Unlock()
	return v
}


// as Put()
//   this simplifies $a.Put "key" $value | printf ""
//   since Put returns $value
func (m *Map) PutReturnBlank(k string, v interface{}) interface{} {
	m.mu.Lock()
	m.data[k] = v
	m.mu.Unlock()
	return ""
}

// void  putAll(Map<? extends K,? extends V> m)
//    Copies all of the mappings from the specified map to this map (optional operation).
func (m *Map) PutAll(m2 Map) {
	m.mu.Lock()
	for k := range m2.data {
		m.data[k] = m2.data[k]
	}
	m.mu.Unlock()
}

// as PutAll
// Template requires a return, this version returns data
func (m *Map) PutAllReturnData(m2 Map) map[string]interface{} {
	m.PutAll(m2)

	m.mu.RLock()
	rt := m.data
	m.mu.RUnlock()

	return rt
}

// as PutAll
// Template requires a return, this version returns blank
func (m Map) PutAllReturnBlank (m2 Map) string {
	m.PutAll(m2)
	return ""
}

// V   remove(Object key)
//     Removes the mapping for a key from this map if it is present (optional operation).
func (m *Map) Remove(k string) interface{} {
	m.mu.Lock()
	var rt = m.data[k]
	delete(m.data, k)
	m.mu.Unlock()
	return rt
}

// as remove()
//   simplifies $m.Remove "key" | printf ""
//   since the interface returns $value
func (m *Map) RemoveReturnBlank(k string) string {
	m.Remove(k)
	return ""
}

// int   size()
//    Returns the number of key-value mappings in this map.
func (m *Map) Size() int  {
	m.mu.RLock()
	size := len(m.data)
	m.mu.RUnlock()

	return size
}

// Collection<V>   values()
//    Returns a Collection view of the values contained in this map.
func (m *Map) Values() []interface{} {
	m.mu.RLock()
	values := make([]interface{}, 0, len(m.data))
	for k := range m.data {
		values = append(values, m.data[k])
	}
	m.mu.RUnlock()

	return values
}


// parses a JSON string and replaces the Map data
func (m *Map) ParseJSON(s string) (interface{}, error) {
	if s == "" {
		m.mu.Lock()
		m.data = map[string]interface{}{}
		m.mu.Unlock()
		return m.data, nil
	}

	m.mu.Lock()
	if err := json.Unmarshal([]byte(s), &m.data); err != nil {
		fmt.Printf("err, %+v", err);
		m.mu.Unlock()
		return nil, err
	}
	m.mu.Unlock()

	return m.data, nil
}

// exports the map to a JSON string
func (m *Map) ToJSON() (string, error) {
	m.mu.RLock()
	result, err := json.Marshal(m.data)
	if err != nil {
		fmt.Printf("toJSON: %s", err)
		m.mu.RUnlock()
		return "", fmt.Errorf("toJSON: %s", err)
	}
	m.mu.RUnlock()

	return string(bytes.TrimSpace(result)), err
}

