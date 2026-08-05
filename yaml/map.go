package yaml

import "encoding/json"

// Map is what a YAML mapping parses to: an ordered set of key/value pairs.
// A plain Go map has no iteration order, but YAML mapping order is sometimes
// semantic to a caller (e.g. a caller that assigns meaning to declaration
// order), so Parse/ParseAll/Unmarshal preserve it here instead of discarding
// it into map[string]any.
type Map struct {
	Keys   []string
	Values map[string]any
}

func newMap() *Map {
	return &Map{Values: map[string]any{}}
}

// set inserts or overwrites k. Callers that must reject duplicate keys check
// Get first; set itself is silent about it (parseMapping is the one caller,
// and it already errors on a duplicate before calling set).
func (m *Map) set(k string, v any) {
	if _, exists := m.Values[k]; !exists {
		m.Keys = append(m.Keys, k)
	}
	m.Values[k] = v
}

// Get returns the value for k and whether it was present.
func (m *Map) Get(k string) (any, bool) {
	v, ok := m.Values[k]
	return v, ok
}

// Len returns the number of entries.
func (m *Map) Len() int {
	return len(m.Keys)
}

// MarshalJSON emits the mapping as a JSON object, in declaration order.
// encoding/json has no notion of ordered maps -- without this, encoding a
// *Map would serialize its exported Keys/Values struct fields literally
// instead of the mapping they represent.
func (m *Map) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	var b []byte
	b = append(b, '{')
	for i, k := range m.Keys {
		if i > 0 {
			b = append(b, ',')
		}
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		b = append(b, kb...)
		b = append(b, ':')
		vb, err := json.Marshal(m.Values[k])
		if err != nil {
			return nil, err
		}
		b = append(b, vb...)
	}
	b = append(b, '}')
	return b, nil
}
