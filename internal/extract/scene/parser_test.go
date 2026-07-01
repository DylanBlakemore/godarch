package scene

import "testing"

const sampleScene = `[gd_scene load_steps=2 format=3 uid="uid://abc"]

[ext_resource type="Script" path="res://player.gd" id="1_player"]

[node name="Main" type="Node"]
script = ExtResource("1_player")

[node name="Timer" type="Timer" parent="."]

[connection signal="timeout" from="Timer" to="." method="_on_timer_timeout"]
`

func TestParseSectionsAndAttrs(t *testing.T) {
	doc := Parse("res://main.tscn", []byte(sampleScene))

	var types []string
	for _, s := range doc.Sections {
		types = append(types, s.Type)
	}
	want := []string{"gd_scene", "ext_resource", "node", "node", "connection"}
	if len(types) != len(want) {
		t.Fatalf("section types = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("section %d = %q, want %q", i, types[i], want[i])
		}
	}

	ext := doc.Sections[1]
	if s, _ := ext.Attrs["type"].AsString(); s != "Script" {
		t.Errorf("ext type = %q", s)
	}
	if s, _ := ext.Attrs["path"].AsString(); s != "res://player.gd" {
		t.Errorf("ext path = %q", s)
	}

	main := doc.Sections[2]
	if !main.Props["script"].IsCtor("ExtResource") {
		t.Errorf("main script prop not ExtResource: %+v", main.Props["script"])
	}
	if main.PropLines["script"] != 6 {
		t.Errorf("script prop line = %d, want 6", main.PropLines["script"])
	}

	conn := doc.Sections[4]
	if s, _ := conn.Attrs["signal"].AsString(); s != "timeout" {
		t.Errorf("connection signal = %q", s)
	}
	if s, _ := conn.Attrs["method"].AsString(); s != "_on_timer_timeout" {
		t.Errorf("connection method = %q", s)
	}
}

func TestParseMultilineValue(t *testing.T) {
	// A multi-line dict value (as in project.godot input actions) must be
	// accumulated into a single value rather than truncated.
	input := "[input]\n\njump={\n\"deadzone\": 0.5,\n\"events\": []\n}\n"
	doc := Parse("project.godot", []byte(input))
	sec := doc.Sections[len(doc.Sections)-1]
	v, ok := sec.Props["jump"]
	if !ok {
		t.Fatalf("jump prop missing; sections=%+v", doc.Sections)
	}
	if v.Type != ValDict {
		t.Fatalf("jump value type = %v, want dict", v.Type)
	}
	if len(doc.Diags) != 0 {
		t.Errorf("unexpected diagnostics: %v", doc.Diags)
	}
}

func TestParseLenientBadValue(t *testing.T) {
	// An unparseable value must not fail the whole file: a diagnostic is emitted
	// and the raw text kept as a string (M1 exit criterion: no silent drops).
	doc := Parse("res://x.tres", []byte("[resource]\nfoo = )bad(\n"))
	if len(doc.Diags) == 0 {
		t.Fatalf("expected a diagnostic for the bad value")
	}
	sec := doc.Sections[0]
	if _, ok := sec.Props["foo"]; !ok {
		t.Errorf("bad prop dropped instead of kept as raw")
	}
}

func TestIsBinaryGodot(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"uncompressed magic", "RSRC\x00\x00\x00\x00binary junk", true},
		{"compressed magic", "RSCC\x01\x02\x03\x04", true},
		{"text scene header", sampleScene, false},
		{"text resource header", "[gd_resource type=\"Resource\" format=3]\n", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBinaryGodot([]byte(tt.data)); got != tt.want {
				t.Errorf("isBinaryGodot(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
