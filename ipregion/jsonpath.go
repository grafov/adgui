package ipregion

import (
	"encoding/json"
	"fmt"
	"strings"
)

func jsonPath(data []byte, path string) string {
	path = strings.TrimSpace(path)
	if path == "" || len(data) == 0 {
		return ""
	}
	if path[0] != '.' {
		path = "." + path
	}

	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}

	val := walkJSONPath(root, strings.Split(path[1:], "."))
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strings.TrimSpace(fmt.Sprintf("%d", int64(v)))
		}
		return strings.TrimSpace(fmt.Sprintf("%g", v))
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		s := strings.TrimSpace(string(b))
		if s == "null" {
			return ""
		}
		return strings.TrimSpace(strings.Trim(s, `"`))
	}
}

func walkJSONPath(node any, parts []string) any {
	if len(parts) == 0 {
		return node
	}
	part := parts[0]
	rest := parts[1:]

	switch cur := node.(type) {
	case map[string]any:
		if v, ok := cur[part]; ok {
			return walkJSONPath(v, rest)
		}
		return nil
	case []any:
		if part == "" {
			return nil
		}
		idx := 0
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || part[i] > '9' {
				return nil
			}
			idx = idx*10 + int(part[i]-'0')
		}
		if idx < 0 || idx >= len(cur) {
			return nil
		}
		return walkJSONPath(cur[idx], rest)
	default:
		return nil
	}
}

func isValidJSON(data []byte) bool {
	var v any
	return json.Unmarshal(data, &v) == nil
}
