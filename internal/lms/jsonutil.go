package lms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func decodeObject(data []byte) (any, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func normalizedKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func lookup(m map[string]any, aliases ...string) (any, bool) {
	wanted := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		wanted[normalizedKey(alias)] = struct{}{}
	}
	for k, v := range m {
		if _, ok := wanted[normalizedKey(k)]; ok {
			return v, true
		}
	}
	return nil, false
}

func lookupRecursive(v any, aliases ...string) (any, bool) {
	switch x := v.(type) {
	case map[string]any:
		if found, ok := lookup(x, aliases...); ok {
			return found, true
		}
		for _, child := range x {
			if found, ok := lookupRecursive(child, aliases...); ok {
				return found, true
			}
		}
	case []any:
		for _, child := range x {
			if found, ok := lookupRecursive(child, aliases...); ok {
				return found, true
			}
		}
	}
	return nil, false
}

func asString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, x != ""
	case json.Number:
		return x.String(), true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(x), true
	default:
		return "", false
	}
}

func asFloat(v any) (float64, bool) {
	var f float64
	var err error
	switch x := v.(type) {
	case json.Number:
		f, err = x.Float64()
	case float64:
		f = x
	case float32:
		f = float64(x)
	case int:
		f = float64(x)
	case int64:
		f = float64(x)
	case string:
		f, err = strconv.ParseFloat(strings.TrimSpace(x), 64)
	default:
		return 0, false
	}
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func asBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(x))
		if err == nil {
			return b, true
		}
		status := strings.ToLower(strings.TrimSpace(x))
		if status == "generating" || status == "running" || status == "active" {
			return true, true
		}
		if status == "idle" || status == "stopped" || status == "not-running" || status == "inactive" {
			return false, true
		}
	case json.Number:
		f, err := x.Float64()
		if err == nil {
			return f != 0, true
		}
	case float64:
		return x != 0, true
	}
	return false, false
}

func firstString(m map[string]any, aliases ...string) string {
	if v, ok := lookup(m, aliases...); ok {
		if s, ok := asString(v); ok {
			return s
		}
	}
	return ""
}

func firstFloat(m map[string]any, aliases ...string) (float64, bool) {
	if v, ok := lookup(m, aliases...); ok {
		return asFloat(v)
	}
	return 0, false
}

func firstFloatRecursive(v any, aliases ...string) (float64, bool) {
	if found, ok := lookupRecursive(v, aliases...); ok {
		return asFloat(found)
	}
	return 0, false
}

func parseHumanBytes(v any) (float64, bool) {
	if f, ok := asFloat(v); ok {
		return f, true
	}
	s, ok := asString(v)
	if !ok {
		return 0, false
	}
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) == 0 {
		return 0, false
	}
	n, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, false
	}
	if len(parts) == 1 {
		return n, true
	}
	unit := strings.ToUpper(strings.TrimSpace(parts[1]))
	multipliers := map[string]float64{
		"B":  1,
		"KB": 1e3, "KIB": 1024,
		"MB": 1e6, "MIB": 1024 * 1024,
		"GB": 1e9, "GIB": 1024 * 1024 * 1024,
		"TB": 1e12, "TIB": 1024 * 1024 * 1024 * 1024,
	}
	mult, ok := multipliers[unit]
	if !ok {
		return 0, false
	}
	return n * mult, true
}

func debugShape(v any) string {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		return fmt.Sprintf("object keys=%v", keys)
	case []any:
		return fmt.Sprintf("array len=%d", len(x))
	default:
		return fmt.Sprintf("%T", v)
	}
}
