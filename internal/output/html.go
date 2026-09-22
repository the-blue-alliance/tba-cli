package output

import (
	"html"
	"regexp"
	"strings"
)

// htmlTagPattern matches a single HTML tag. TBA's prose fields only ever carry
// simple inline markup (<b>, <br/>), so a regexp is enough here and saves
// pulling in a parser.
var htmlTagPattern = regexp.MustCompile(`</?([a-zA-Z][a-zA-Z0-9]*)[^>]*>`)

// breakingTags are the tags that separate words rather than decorate them, and
// so leave a space behind when they are removed.
var breakingTags = map[string]bool{
	"br": true, "p": true, "div": true, "li": true, "tr": true, "hr": true,
}

// StripHTML turns one of TBA's HTML status strings into a single line of plain
// text: tags are removed, entities are decoded, and runs of whitespace collapse
// to one space.
//
// TBA sends overall_status_str as marked-up prose — "Team 177 is <b>Rank 1</b>
// with a record of <b>10-2-0</b>." — which would otherwise leak tags into a
// table cell. Inline tags leave nothing behind, so the surrounding punctuation
// stays tight; line-breaking tags leave a space.
func StripHTML(s string) string {
	stripped := htmlTagPattern.ReplaceAllStringFunc(s, func(tag string) string {
		name := strings.ToLower(htmlTagPattern.FindStringSubmatch(tag)[1])
		if breakingTags[name] {
			return " "
		}
		return ""
	})
	return strings.Join(strings.Fields(html.UnescapeString(stripped)), " ")
}
