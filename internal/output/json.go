package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/itchyny/gojq"
)

func PrintJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func PrintJSONWithFilter(data interface{}, jqExpr string) error {
	if jqExpr == "" {
		return PrintJSON(data)
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
		fmt.Println(string(out))
	}
	return nil
}
