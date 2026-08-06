package sql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"
)

// timeLayouts are tried in order when normalizing driver output to time.Time (most precise first).
var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999",
	"2006-01-02 15:04:05.999",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05.999999 -0700",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02",
}

// normalizeRow converts driver-specific types in place: []byte → string or time.Time (if it parses as a datetime).
// Some drivers (e.g. MySQL text protocol) return []byte for string/date columns; limen expects string or time.Time.
func normalizeRow(m map[string]any) {
	for k, v := range m {
		switch x := v.(type) {
		case []byte:
			if t := parseTimeBytes(x); !t.IsZero() {
				m[k] = t
			} else {
				m[k] = string(x)
			}
		case string:
			if t := parseTimeString(x); !t.IsZero() {
				m[k] = t
			}
		}
	}
}

func parseTimeBytes(b []byte) time.Time {
	if len(b) == 0 {
		return time.Time{}
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, string(b)); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseTimeString(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// sortedKeys returns map keys sorted so INSERT/UPDATE column order is deterministic.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// normalizeWriteMap returns a shallow copy with map/slice values JSON-encoded so
// database/sql drivers can bind them (e.g. for JSON/JSONB columns).
func normalizeWriteMap(m map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(m))
	for k, v := range m {
		normalized, err := normalizeWriteValue(v)
		if err != nil {
			return nil, fmt.Errorf("column %q: %w", k, err)
		}
		out[k] = normalized
	}
	return out, nil
}

func normalizeWriteValue(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	// []byte is a slice but must stay raw for drivers (json.Marshal would base64 it).
	if _, ok := v.([]byte); ok {
		return v, nil
	}

	switch reflect.ValueOf(v).Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return string(b), nil
	default:
		return v, nil
	}
}
