package output

import "testing"

func TestStripHTML(t *testing.T) {
	cases := map[string]string{
		"": "",
		"Team 177 was <b>Rank 1</b> with a record of <b>10-2-0</b> in quals.": "Team 177 was Rank 1 with a record of 10-2-0 in quals.",
		// Inline tags leave no space behind, so the sentence keeps its shape.
		"competed as the <b>Captain</b> of <b>Alliance 1</b>, and <b>won the event</b>.": "competed as the Captain of Alliance 1, and won the event.",
		// Line breaks and stray whitespace collapse to single spaces.
		"Rank 1<br/>Record 10-2-0":     "Rank 1 Record 10-2-0",
		"  Rank\n1   with\t12 played ": "Rank 1 with 12 played",
		// Entities are decoded.
		"Gund Foundation &amp; RTX": "Gund Foundation & RTX",
		"5 &lt; 6":                  "5 < 6",
		// A bare angle bracket is not a tag.
		"score > 100": "score > 100",
		// Attributes inside a tag do not leak.
		`<a href="https://thebluealliance.com">TBA</a>`: "TBA",
	}
	for in, want := range cases {
		if got := StripHTML(in); got != want {
			t.Errorf("StripHTML(%q) = %q, want %q", in, got, want)
		}
	}
}
