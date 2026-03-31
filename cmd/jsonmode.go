package cmd

import (
	"encoding/json"
	"os"
)

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// jsonEmit writes one JSON object to stdout (newline-terminated).
func jsonEmit(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func jsonResultOK(command string, fields map[string]interface{}) error {
	m := make(map[string]interface{}, len(fields)+2)
	m["ok"] = true
	m["command"] = command
	for k, v := range fields {
		m[k] = v
	}
	return jsonEmit(m)
}
