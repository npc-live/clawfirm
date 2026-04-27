package main

import "encoding/json"

// jsonStr returns a JSON-encoded string literal (with quotes).
func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// jsonMarshal marshals v to a JSON byte slice, ignoring errors.
func jsonMarshal(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

// jsonUnmarshal is a thin wrapper for json.Unmarshal.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
