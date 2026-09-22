package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"
)

// PrintJSON writes data as indented JSON followed by a newline.
func PrintJSON(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// PrintJSONWithFilter writes data as indented JSON, optionally passing it
// through a jq expression first. Each jq result is written on its own line.
func PrintJSONWithFilter(w io.Writer, data interface{}, jqExpr string) error {
	if jqExpr == "" {
		return PrintJSON(w, data)
	}

	query, err := gojq.Parse(jqExpr)
	if err != nil {
		return fmt.Errorf("invalid jq expression: %w", err)
	}

	// Convert to interface{} via JSON round-trip
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	iter := query.Run(v)
	for {
		val, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := val.(error); ok {
			return err
		}
		out, err := json.MarshalIndent(val, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(out))
	}
	return nil
}
