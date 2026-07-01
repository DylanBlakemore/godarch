package scene

import (
	"fmt"
	"strconv"
	"strings"
)

// ValueType tags a parsed Godot literal. Scenes/resources store property values
// in a small literal grammar (DESIGN §3.3): scalars, arrays, dicts, and typed
// constructors such as ExtResource("id"), SubResource("id"), NodePath("…"), and
// Vector3(…). Values are parsed into a Value AST the extractor walks.
type ValueType int

const (
	ValString ValueType = iota // "…"
	ValInt                     // 42, -1
	ValFloat                   // 0.5, 3.0e2
	ValBool                    // true / false
	ValNull                    // null
	ValIdent                   // a bareword (enum constant, class name)
	ValArray                   // [ … ]
	ValDict                    // { "k": v, … }
	ValCtor                    // Name( … ) — includes ExtResource/SubResource/NodePath
)

// Pair is one entry of a ValDict.
type Pair struct {
	Key Value
	Val Value
}

// Value is a parsed Godot literal. Only the fields relevant to its Type are set.
type Value struct {
	Type  ValueType
	Str   string  // ValString content (unescaped) or ValIdent name
	Int   int64   // ValInt
	Float float64 // ValFloat
	Bool  bool    // ValBool
	Name  string  // ValCtor constructor name (ExtResource, Vector3, …)
	Args  []Value // ValCtor arguments / ValArray elements
	Pairs []Pair  // ValDict entries
	Raw   string  // original source slice, for diagnostics and raw fallbacks
}

// AsString returns the string content of a ValString.
func (v Value) AsString() (string, bool) {
	if v.Type == ValString {
		return v.Str, true
	}
	return "", false
}

// IsCtor reports whether v is a constructor with the given name.
func (v Value) IsCtor(name string) bool {
	return v.Type == ValCtor && v.Name == name
}

// RefID returns the id argument of an ExtResource/SubResource reference. Godot
// has spelled the id as both a string ("1_player") and, in older formats, a
// bare number (1); either collapses to the same key via scalarString.
func (v Value) RefID() (string, bool) {
	if v.Type != ValCtor || (v.Name != "ExtResource" && v.Name != "SubResource") {
		return "", false
	}
	if len(v.Args) == 0 {
		return "", false
	}
	return v.Args[0].scalarString(), true
}

// StringItems returns the string elements of a ValArray, skipping non-strings.
// Used for groups=["a","b"] and similar homogeneous string arrays.
func (v Value) StringItems() []string {
	if v.Type != ValArray {
		return nil
	}
	out := make([]string, 0, len(v.Args))
	for _, a := range v.Args {
		if s, ok := a.AsString(); ok {
			out = append(out, s)
		}
	}
	return out
}

// scalarString renders a scalar value as a plain string (for ids and NodePath
// bodies). Non-scalars fall back to their raw source.
func (v Value) scalarString() string {
	switch v.Type {
	case ValString, ValIdent:
		return v.Str
	case ValInt:
		return strconv.FormatInt(v.Int, 10)
	case ValFloat:
		return strconv.FormatFloat(v.Float, 'g', -1, 64)
	case ValBool:
		if v.Bool {
			return "true"
		}
		return "false"
	default:
		return v.Raw
	}
}

// ParseValue parses a single Godot literal, requiring it to consume the whole
// (trimmed) input. Callers that tolerate malformed values recover from the error
// and keep the raw text.
func ParseValue(s string) (Value, error) {
	sc := &scanner{s: s}
	sc.skipWS()
	v, err := sc.value()
	if err != nil {
		return Value{}, err
	}
	sc.skipWS()
	if !sc.eof() {
		return Value{}, fmt.Errorf("trailing text %q", sc.s[sc.i:])
	}
	return v, nil
}

// scanner is a cursor over a value string for recursive-descent parsing.
type scanner struct {
	s string
	i int
}

func (sc *scanner) eof() bool { return sc.i >= len(sc.s) }

func (sc *scanner) skipWS() {
	for !sc.eof() {
		switch sc.s[sc.i] {
		case ' ', '\t', '\n', '\r':
			sc.i++
		default:
			return
		}
	}
}

func (sc *scanner) value() (Value, error) {
	sc.skipWS()
	if sc.eof() {
		return Value{}, fmt.Errorf("unexpected end of value")
	}
	start := sc.i
	c := sc.s[sc.i]
	switch {
	case c == '"':
		return sc.stringLit(start)
	case c == '[':
		return sc.array(start)
	case c == '{':
		return sc.dict(start)
	case c == '-' || c == '+' || (c >= '0' && c <= '9'):
		return sc.number()
	case isIdentStart(c):
		return sc.identOrCtor(start)
	default:
		return Value{}, fmt.Errorf("unexpected char %q", string(c))
	}
}

func (sc *scanner) stringLit(start int) (Value, error) {
	sc.i++ // opening quote
	var b strings.Builder
	for !sc.eof() {
		c := sc.s[sc.i]
		switch c {
		case '\\':
			sc.i++
			if sc.eof() {
				return Value{}, fmt.Errorf("unterminated string escape")
			}
			switch e := sc.s[sc.i]; e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte(e)
			}
			sc.i++
		case '"':
			sc.i++
			return Value{Type: ValString, Str: b.String(), Raw: sc.s[start:sc.i]}, nil
		default:
			b.WriteByte(c)
			sc.i++
		}
	}
	return Value{}, fmt.Errorf("unterminated string")
}

func (sc *scanner) number() (Value, error) {
	j := sc.i
	if sc.s[j] == '+' || sc.s[j] == '-' {
		j++
	}
	isFloat := false
	for j < len(sc.s) {
		c := sc.s[j]
		switch {
		case c >= '0' && c <= '9':
			j++
		case c == '.' || c == 'e' || c == 'E':
			isFloat = true
			j++
		case (c == '+' || c == '-') && (sc.s[j-1] == 'e' || sc.s[j-1] == 'E'):
			j++
		default:
			c = 0 // sentinel: stop
		}
		if c == 0 {
			break
		}
	}
	tok := sc.s[sc.i:j]
	sc.i = j
	if !isFloat {
		if n, err := strconv.ParseInt(tok, 10, 64); err == nil {
			return Value{Type: ValInt, Int: n, Raw: tok}, nil
		}
	}
	f, err := strconv.ParseFloat(tok, 64)
	if err != nil {
		return Value{}, fmt.Errorf("bad number %q", tok)
	}
	return Value{Type: ValFloat, Float: f, Raw: tok}, nil
}

func (sc *scanner) identOrCtor(start int) (Value, error) {
	j := sc.i
	for j < len(sc.s) && isIdentPart(sc.s[j]) {
		j++
	}
	name := sc.s[sc.i:j]
	sc.i = j
	if !sc.eof() && sc.s[sc.i] == '(' {
		args, err := sc.argList('(', ')')
		if err != nil {
			return Value{}, err
		}
		return Value{Type: ValCtor, Name: name, Args: args, Raw: sc.s[start:sc.i]}, nil
	}
	switch name {
	case "true":
		return Value{Type: ValBool, Bool: true, Raw: name}, nil
	case "false":
		return Value{Type: ValBool, Bool: false, Raw: name}, nil
	case "null":
		return Value{Type: ValNull, Raw: name}, nil
	}
	return Value{Type: ValIdent, Str: name, Raw: name}, nil
}

// argList parses a comma-separated list until the matching close byte. A colon
// is treated as an additional separator so Godot's Object("key":val,…) literal
// parses leniently without the extractor needing to model its internals.
func (sc *scanner) argList(open, closer byte) ([]Value, error) {
	sc.i++ // consume open
	var args []Value
	for {
		sc.skipWS()
		if sc.eof() {
			return nil, fmt.Errorf("unterminated %q", string(open))
		}
		if sc.s[sc.i] == closer {
			sc.i++
			return args, nil
		}
		v, err := sc.value()
		if err != nil {
			return nil, err
		}
		args = append(args, v)
		sc.skipWS()
		if sc.eof() {
			return nil, fmt.Errorf("unterminated %q", string(open))
		}
		switch sc.s[sc.i] {
		case ',', ':':
			sc.i++
		case closer:
			sc.i++
			return args, nil
		default:
			return nil, fmt.Errorf("expected ',' or %q, got %q", string(closer), string(sc.s[sc.i]))
		}
	}
}

func (sc *scanner) array(start int) (Value, error) {
	args, err := sc.argList('[', ']')
	if err != nil {
		return Value{}, err
	}
	return Value{Type: ValArray, Args: args, Raw: sc.s[start:sc.i]}, nil
}

func (sc *scanner) dict(start int) (Value, error) {
	sc.i++ // consume {
	var pairs []Pair
	for {
		sc.skipWS()
		if sc.eof() {
			return Value{}, fmt.Errorf("unterminated dict")
		}
		if sc.s[sc.i] == '}' {
			sc.i++
			break
		}
		key, err := sc.value()
		if err != nil {
			return Value{}, err
		}
		sc.skipWS()
		if sc.eof() || sc.s[sc.i] != ':' {
			return Value{}, fmt.Errorf("expected ':' in dict")
		}
		sc.i++ // consume :
		val, err := sc.value()
		if err != nil {
			return Value{}, err
		}
		pairs = append(pairs, Pair{Key: key, Val: val})
		sc.skipWS()
		if sc.eof() {
			return Value{}, fmt.Errorf("unterminated dict")
		}
		switch sc.s[sc.i] {
		case ',':
			sc.i++
		case '}':
			sc.i++
			return Value{Type: ValDict, Pairs: pairs, Raw: sc.s[start:sc.i]}, nil
		default:
			return Value{}, fmt.Errorf("expected ',' or '}' in dict")
		}
	}
	return Value{Type: ValDict, Pairs: pairs, Raw: sc.s[start:sc.i]}, nil
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
