package template

import (
  "sync"
  "encoding/json"
  "bytes"
  "fmt"
)

type Map struct {
    sync.RWMutex

    // data is the map of everything
    data map[string]interface{}

}

func NewMap() *Map {
  return &Map{
    data: make(map[string]interface{}),
  }
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


func (m Map) Clear () {
  for k := range m.data {
    delete(m.data, k)
  }

  // m.data = make(map[string]interface{}) // not working?
}

func (m Map) ClearReturnData () map[string]interface{}{
  m.Clear()

  return m.data
}

func (m Map) ClearReturnBlank () string {
  m.Clear()
  return ""
}

func (m *Map) ContainsKey(k string) bool {
  if _, ok := m.data[k]; ok {
    return true
  }
  return false
}

func (m *Map) ContainsValue(v interface{}) bool {
  for k := range m.data {
    if v ==  m.data[k] {
      return true
    }
  }
  return false
}

/*
func (m *Map) EntrySet() []interface{} {
  // not yet implemented
}

func (m *Map) Equals(interface{}) bool {
  // not yet implemented
}
*/

func (m *Map) Get(k string) interface{} {
  return m.data[k]
}

/*
func (m *Map) HashCode() int {
  // not yet implemented
}
*/

func (m *Map) IsEmpty() bool {
  if len(m.data) > 0 {
    return false
  }
  return true
}

func (m *Map) KeySet() []string {
  keys := make([]string, 0, len(m.data))
  for k := range m.data {
    keys = append(keys, k)
  }
  return keys
}

func (m *Map) Put(k string, v interface{}) interface{} {
  m.data[k] = v
  return v
}

func (m *Map) PutAll(m2 Map) {
  for k := range m2.data {
    m.data[k] = m2.data[k]
  }
}

func (m *Map) PutAllReturnData(m2 Map) map[string]interface{} {
  m.PutAll(m2)

  return m.data
}

func (m Map) PutAllReturnBlank (m2 Map) string {
  m.PutAll(m2)
  return ""
}

func (m *Map) Remove(k string) interface{} {
  var rt = m.data[k]
  delete(m.data, k)
  return rt
}

func (m *Map) Size() int  {
  return len(m.data)
}

func (m *Map) Values() []interface{} {
  values := make([]interface{}, 0, len(m.data))
  for k := range m.data {
    values = append(values, m.data[k])
  }
  return values
}

func (m *Map) ParseJSON(s string) (interface{}, error) {
  if s == "" {
    m.data = map[string]interface{}{}
  }

  if err := json.Unmarshal([]byte(s), &m.data); err != nil {
    fmt.Printf("err, %+v", err);
    return nil, err
  }

  return m.data, nil
}

func (m *Map) ToJSON() (string, error) {
  result, err := json.Marshal(m.data)
  if err != nil {
    fmt.Printf("toJSON: %s", err)
    return "", fmt.Errorf("toJSON: %s", err)
  }
  return string(bytes.TrimSpace(result)), err
}


func (m *Map) HelloWorld() string  {
  return "hello world"
}
