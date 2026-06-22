package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	raw = coerceJSONScalars(raw, dst)
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// coerceScalarStrings rewrites top-level JSON object fields that arrived as
// strings (e.g. a browser form sending {"value":"5000000"}) into the numeric
// or boolean kind the destination struct field expects. Nested struct and
// slice-of-struct fields (e.g. a contract's line-item or milestone arrays) are
// recursed into.
func coerceScalarStrings(t reflect.Type, m map[string]any) {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		val, present := m[name]
		if !present {
			continue
		}
		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if s, ok := val.(string); ok && s == "" {
			switch ft.Kind() {
			case reflect.Float32, reflect.Float64,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Bool:
				// Blank numeric/bool form value clears the field: JSON null decodes
				// to zero (value) or nil (pointer); "" would fail the unmarshal.
				m[name] = nil
				continue
			}
		}
		switch ft.Kind() {
		case reflect.Struct:
			if nested, ok := val.(map[string]any); ok {
				coerceScalarStrings(ft, nested)
			}
		case reflect.Slice, reflect.Array:
			et := ft.Elem()
			for et.Kind() == reflect.Ptr {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				if arr, ok := val.([]any); ok {
					for _, el := range arr {
						if nested, ok := el.(map[string]any); ok {
							coerceScalarStrings(et, nested)
						}
					}
				}
			}
		case reflect.Float32, reflect.Float64:
			if s, ok := val.(string); ok && s != "" {
				if v, err := strconv.ParseFloat(s, 64); err == nil {
					m[name] = v
				}
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if s, ok := val.(string); ok && s != "" {
				if v, err := strconv.ParseInt(s, 10, 64); err == nil {
					m[name] = v
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if s, ok := val.(string); ok && s != "" {
				if v, err := strconv.ParseUint(s, 10, 64); err == nil {
					m[name] = v
				}
			}
		case reflect.Bool:
			if s, ok := val.(string); ok && s != "" {
				if v, err := strconv.ParseBool(s); err == nil {
					m[name] = v
				}
			}
		}
	}
}

// coerceJSONScalars round-trips the raw body through map[string]any (or a slice
// of them) so that string-encoded scalars can be coerced to the destination's
// numeric/bool field kinds before strict decoding. Re-marshalling via a map
// preserves all original keys, so json.DisallowUnknownFields still rejects
// unknown fields on the subsequent decode. If anything fails to parse the
// original bytes are returned unchanged.
func coerceJSONScalars(raw []byte, dst any) []byte {
	t := reflect.TypeOf(dst)
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return raw
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		var rows []map[string]any
		if json.Unmarshal(raw, &rows) != nil {
			return raw
		}
		et := t.Elem()
		for _, m := range rows {
			coerceScalarStrings(et, m)
		}
		if out, err := json.Marshal(rows); err == nil {
			return out
		}
	case reflect.Struct:
		var m map[string]any
		if json.Unmarshal(raw, &m) != nil {
			return raw
		}
		coerceScalarStrings(t, m)
		if out, err := json.Marshal(m); err == nil {
			return out
		}
	}
	return raw
}
