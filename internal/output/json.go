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

// ValidateJQ reports whether expr is a jq program at all.
//
// It is separate from running the program so that a misspelled expression can
// be caught before any work is done: an expression that does not parse is a
// mistake in the command line, while one that fails on the data is a failure of
// the run, and the two deserve different exit codes.
func ValidateJQ(expr string) error {
	if expr == "" {
		return nil
	}
	if _, err := gojq.Parse(expr); err != nil {
		return fmt.Errorf("invalid jq expression %q: %w", expr, err)
	}
	return nil
}

// PrintJSONWithFilter writes data as JSON, optionally passing it through a jq
// expression first.
//
// Without a jq expression the data is pretty-printed. With one, a single
// result stays pretty-printed and several results are written one compact
// result per line (NDJSON), so the output stays line-oriented. rawOutput is
// jq's -r: string results lose their quotes, and other values are written
// compactly on a single line.
func PrintJSONWithFilter(w io.Writer, data interface{}, jqExpr string, rawOutput bool) error {
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

	var results []interface{}
	iter := query.Run(v)
	for {
		val, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := val.(error); ok {
			return err
		}
		results = append(results, val)
	}

	compact := rawOutput || len(results) > 1
	for _, val := range results {
		if s, ok := val.(string); ok && rawOutput {
			fmt.Fprintln(w, s)
			continue
		}
		var (
			out []byte
			err error
		)
		if compact {
			out, err = json.Marshal(val)
		} else {
			out, err = json.MarshalIndent(val, "", "  ")
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(out))
	}
	return nil
}
