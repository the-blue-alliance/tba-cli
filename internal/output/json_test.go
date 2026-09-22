package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type sampleTeam struct {
	Key        string `json:"key"`
	TeamNumber int    `json:"team_number"`
	Nickname   string `json:"nickname"`
}

func TestPrintJSONIndentsWithTwoSpaces(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintJSON(&buf, sampleTeam{Key: "frc177", TeamNumber: 177, Nickname: "Bobcat Robotics"}); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}

	want := "{\n" +
		"  \"key\": \"frc177\",\n" +
		"  \"team_number\": 177,\n" +
		"  \"nickname\": \"Bobcat Robotics\"\n" +
		"}\n"
	if buf.String() != want {
		t.Errorf("json =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPrintJSONIndentsNestedValues(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{"alliances": map[string]any{"red": map[string]any{"score": 88}}}
	if err := PrintJSON(&buf, data); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}

	want := "{\n" +
		"  \"alliances\": {\n" +
		"    \"red\": {\n" +
		"      \"score\": 88\n" +
		"    }\n" +
		"  }\n" +
		"}\n"
	if buf.String() != want {
		t.Errorf("json =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPrintJSONPassesRawMessageThrough(t *testing.T) {
	var buf bytes.Buffer
	raw := json.RawMessage(`{"qual":{"high_score":88}}`)
	if err := PrintJSON(&buf, raw); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"high_score": 88`) {
		t.Errorf("raw JSON should be re-indented, got %q", buf.String())
	}
	var back map[string]any
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Errorf("output does not round-trip: %v", err)
	}
}

func TestPrintJSONRejectsUnencodableValues(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintJSON(&buf, make(chan int)); err == nil {
		t.Error("want an error encoding a channel")
	}
}

func TestPrintJSONWithFilterEmptyExpressionIsPlainJSON(t *testing.T) {
	var a, b bytes.Buffer
	data := sampleTeam{Key: "frc177", TeamNumber: 177, Nickname: "Bobcat Robotics"}
	if err := PrintJSON(&a, data); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	if err := PrintJSONWithFilter(&b, data, ""); err != nil {
		t.Fatalf("PrintJSONWithFilter: %v", err)
	}
	if a.String() != b.String() {
		t.Errorf("an empty jq expression should be a no-op:\n%q\n%q", a.String(), b.String())
	}
}

func TestPrintJSONWithFilterSelectsAField(t *testing.T) {
	var buf bytes.Buffer
	data := sampleTeam{Key: "frc177", TeamNumber: 177, Nickname: "Bobcat Robotics"}
	if err := PrintJSONWithFilter(&buf, data, ".nickname"); err != nil {
		t.Fatalf("PrintJSONWithFilter: %v", err)
	}
	if buf.String() != "\"Bobcat Robotics\"\n" {
		t.Errorf("jq output = %q", buf.String())
	}
}

func TestPrintJSONWithFilterWritesOneResultPerLine(t *testing.T) {
	var buf bytes.Buffer
	data := []sampleTeam{{Key: "frc177"}, {Key: "frc1073"}}
	if err := PrintJSONWithFilter(&buf, data, ".[].key"); err != nil {
		t.Fatalf("PrintJSONWithFilter: %v", err)
	}
	if buf.String() != "\"frc177\"\n\"frc1073\"\n" {
		t.Errorf("jq output = %q", buf.String())
	}
}

func TestPrintJSONWithFilterIndentsObjectResults(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]any{{"key": "frc177", "rank": 1}}
	if err := PrintJSONWithFilter(&buf, data, ".[0]"); err != nil {
		t.Fatalf("PrintJSONWithFilter: %v", err)
	}
	want := "{\n  \"key\": \"frc177\",\n  \"rank\": 1\n}\n"
	if buf.String() != want {
		t.Errorf("jq output =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPrintJSONWithFilterRejectsAnInvalidExpression(t *testing.T) {
	var buf bytes.Buffer
	err := PrintJSONWithFilter(&buf, map[string]any{}, ".[")
	if err == nil {
		t.Fatal("want a parse error")
	}
	if !strings.Contains(err.Error(), "invalid jq expression") {
		t.Errorf("error = %v", err)
	}
}

func TestPrintJSONWithFilterReportsRuntimeErrors(t *testing.T) {
	var buf bytes.Buffer
	// Indexing a string is a jq runtime error.
	err := PrintJSONWithFilter(&buf, "a string", ".foo")
	if err == nil {
		t.Fatal("want a runtime error")
	}
	if strings.Contains(err.Error(), "invalid jq expression") {
		t.Errorf("runtime errors should not be reported as parse errors: %v", err)
	}
}

func TestPrintJSONWithFilterOnUnmarshalableData(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintJSONWithFilter(&buf, make(chan int), "."); err == nil {
		t.Error("want an error marshalling a channel")
	}
}
