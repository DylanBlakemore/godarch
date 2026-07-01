package scene

import (
	"regexp"
	"strings"
)

// Section is one INI-style block: a header ([type attr=val …]) and the property
// lines (key = value) that follow it until the next header.
type Section struct {
	Type      string           // header type: gd_scene, ext_resource, node, connection, resource, …
	Attrs     map[string]Value // header attributes
	Props     map[string]Value // body key = value lines
	PropLines map[string]int   // 1-based source line of each property (for evidence)
	Line      int              // 1-based source line of the header
}

// Diagnostic records a value godarch could not parse. Nothing is dropped
// silently (M1 exit criterion): the raw text is kept and the problem surfaced.
type Diagnostic struct {
	File string
	Line int
	Msg  string
}

// Document is a parsed INI-style file: its ordered sections plus any
// diagnostics. Section order matches the file, so extraction is deterministic.
type Document struct {
	Sections []*Section
	Diags    []Diagnostic
}

// keyLineRe matches a body "key = value" line. The key may contain '/' and '.'
// so project.godot keys (config/features, application/config) also match.
var keyLineRe = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_/.]*)\s*=(.*)$`)

// Parse tokenises a Godot INI-style file (.tscn/.tres/.import/project.godot/
// plugin.cfg/.gdextension) into sections. It is lenient: an unparseable value
// yields a diagnostic and is kept as a raw string; a malformed line is skipped.
// Multi-line values (arrays/dicts spanning lines) are accumulated until their
// brackets and quotes balance.
func Parse(file string, data []byte) *Document {
	doc := &Document{}
	lines := strings.Split(string(data), "\n")
	var cur *Section

	for i := 0; i < len(lines); {
		raw := lines[i]
		trimmed := strings.TrimSpace(raw)
		lineNo := i + 1

		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "[") {
			hdr, next := accumulate(lines, i, balanced)
			cur = parseHeader(file, hdr, lineNo, doc)
			doc.Sections = append(doc.Sections, cur)
			i = next
			continue
		}

		m := keyLineRe.FindStringSubmatch(raw)
		if m == nil {
			i++
			continue
		}
		key, valStr := m[1], m[2]
		if !balanced(valStr) {
			valStr, i = accumulateValue(lines, i, valStr)
		} else {
			i++
		}

		if cur == nil {
			// Preamble props (e.g. project.godot's config_version=5 before any
			// section) attach to a synthetic typeless section.
			cur = newSection("", lineNo)
			doc.Sections = append(doc.Sections, cur)
		}
		v, err := ParseValue(strings.TrimSpace(valStr))
		if err != nil {
			doc.Diags = append(doc.Diags, Diagnostic{
				File: file, Line: lineNo,
				Msg: "unparseable value for " + key + ": " + err.Error(),
			})
			v = Value{Type: ValString, Str: strings.TrimSpace(valStr), Raw: valStr}
		}
		cur.Props[key] = v
		cur.PropLines[key] = lineNo
	}
	return doc
}

func newSection(typ string, line int) *Section {
	return &Section{
		Type:      typ,
		Attrs:     map[string]Value{},
		Props:     map[string]Value{},
		PropLines: map[string]int{},
		Line:      line,
	}
}

// accumulate joins physical lines starting at i until done(joined) is true or
// the file ends, returning the joined text and the index of the next line.
func accumulate(lines []string, i int, done func(string) bool) (string, int) {
	joined := lines[i]
	j := i
	for !done(joined) && j+1 < len(lines) {
		j++
		joined += "\n" + lines[j]
	}
	return joined, j + 1
}

// accumulateValue joins continuation lines of a multi-line value onto valStr
// (the text after '=' on line i), returning the joined value and the next index.
func accumulateValue(lines []string, i int, valStr string) (string, int) {
	j := i
	for !balanced(valStr) && j+1 < len(lines) {
		j++
		valStr += "\n" + lines[j]
	}
	return valStr, j + 1
}

// parseHeader parses "[type key=val …]" into a Section. Attribute values use the
// same literal grammar as bodies (a node header can carry instance=ExtResource).
func parseHeader(file, hdr string, lineNo int, doc *Document) *Section {
	sec := newSection("", lineNo)
	inner := strings.TrimSpace(hdr)
	inner = strings.TrimPrefix(inner, "[")
	inner = strings.TrimSuffix(strings.TrimSpace(inner), "]")

	sc := &scanner{s: inner}
	sc.skipWS()
	start := sc.i
	for !sc.eof() && isIdentPart(sc.s[sc.i]) {
		sc.i++
	}
	sec.Type = inner[start:sc.i]

	for {
		sc.skipWS()
		if sc.eof() {
			break
		}
		ks := sc.i
		for !sc.eof() && (isIdentPart(sc.s[sc.i]) || sc.s[sc.i] == '/') {
			sc.i++
		}
		key := inner[ks:sc.i]
		if key == "" {
			break
		}
		sc.skipWS()
		if sc.eof() || sc.s[sc.i] != '=' {
			doc.Diags = append(doc.Diags, Diagnostic{
				File: file, Line: lineNo,
				Msg: "malformed header attribute near " + key,
			})
			break
		}
		sc.i++ // consume =
		v, err := sc.value()
		if err != nil {
			doc.Diags = append(doc.Diags, Diagnostic{
				File: file, Line: lineNo,
				Msg: "bad header attribute " + key + ": " + err.Error(),
			})
			break
		}
		sec.Attrs[key] = v
	}
	return sec
}

// balanced reports whether every bracket/brace/paren in s is closed and no
// string is left open — i.e. s is a complete header or value. Brackets inside
// strings are ignored.
func balanced(s string) bool {
	depth := 0
	inStr := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		}
	}
	return depth <= 0 && !inStr
}
