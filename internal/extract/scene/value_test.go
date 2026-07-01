package scene

import "testing"

func TestParseValueScalars(t *testing.T) {
	cases := []struct {
		in   string
		want ValueType
	}{
		{`"hello"`, ValString},
		{`42`, ValInt},
		{`-1`, ValInt},
		{`0.5`, ValFloat},
		{`3.0e2`, ValFloat},
		{`true`, ValBool},
		{`false`, ValBool},
		{`null`, ValNull},
		{`Node2D`, ValIdent},
	}
	for _, c := range cases {
		v, err := ParseValue(c.in)
		if err != nil {
			t.Fatalf("ParseValue(%q): %v", c.in, err)
		}
		if v.Type != c.want {
			t.Errorf("ParseValue(%q).Type = %v, want %v", c.in, v.Type, c.want)
		}
	}
}

func TestParseValueStringUnescape(t *testing.T) {
	v, err := ParseValue(`"a\"b\n"`)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := v.AsString(); got != "a\"b\n" {
		t.Errorf("string = %q, want %q", got, "a\"b\n")
	}
}

func TestParseValueExtResource(t *testing.T) {
	v, err := ParseValue(`ExtResource("1_player")`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsCtor("ExtResource") {
		t.Fatalf("not an ExtResource ctor: %+v", v)
	}
	id, ok := v.RefID()
	if !ok || id != "1_player" {
		t.Errorf("RefID = %q,%v want 1_player,true", id, ok)
	}
}

func TestParseValueNumericRefID(t *testing.T) {
	// Older Godot format used numeric ids: ExtResource( 1 ).
	v, err := ParseValue(`ExtResource( 1 )`)
	if err != nil {
		t.Fatal(err)
	}
	id, ok := v.RefID()
	if !ok || id != "1" {
		t.Errorf("RefID = %q,%v want 1,true", id, ok)
	}
}

func TestParseValueNodePath(t *testing.T) {
	v, err := ParseValue(`NodePath("../Player")`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsCtor("NodePath") {
		t.Fatalf("not a NodePath: %+v", v)
	}
	if v.Args[0].scalarString() != "../Player" {
		t.Errorf("path = %q", v.Args[0].scalarString())
	}
}

func TestParseValueArrayOfStrings(t *testing.T) {
	v, err := ParseValue(`["a", "b", "c"]`)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != ValArray {
		t.Fatalf("type = %v", v.Type)
	}
	got := v.StringItems()
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("items = %v", got)
	}
}

func TestParseValueCtorArgs(t *testing.T) {
	v, err := ParseValue(`Vector3(1, 2, 3)`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsCtor("Vector3") || len(v.Args) != 3 {
		t.Fatalf("bad ctor: %+v", v)
	}
}

func TestParseValueDict(t *testing.T) {
	v, err := ParseValue(`{"deadzone": 0.5, "name": "x"}`)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != ValDict || len(v.Pairs) != 2 {
		t.Fatalf("bad dict: %+v", v)
	}
}

func TestParseValueLenientObject(t *testing.T) {
	// Godot's Object(...) literal mixes positional and key:value args; the parser
	// must consume it without failing (leniency), even if it does not model it.
	in := `Object(InputEventKey,"resource_local_to_scene":false,"keycode":0,"echo":false)`
	if _, err := ParseValue(in); err != nil {
		t.Fatalf("expected lenient parse, got %v", err)
	}
}
