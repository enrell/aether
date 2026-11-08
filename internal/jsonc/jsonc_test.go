package jsonc

import (
	"encoding/json"
	"strings"
	"testing"
)

const terminalSettings = `{
    "$schema": "https://aka.ms/terminal-profiles-schema",
    // keep this comment
    "defaultProfile": "{574e775e-4f2a-5b96-ac1e-a2962a402336}",
    "keybindings":<SP>
    [
        {
            "id": "Terminal.CopyToClipboard",
            "keys": "ctrl+c"
        }
    ],
    "profiles":
    {
        "defaults": {},
        "list":
        [
            {
                "guid": "{574e775e-4f2a-5b96-ac1e-a2962a402336}",
                "name": "PowerShell", // trailing comment
            },
        ]
    },
    "schemes": []
}
`

func mustParse(t *testing.T, s string) *Doc {
	t.Helper()
	// <SP> marks significant trailing whitespace that editors like to strip.
	d, err := Parse([]byte(strings.ReplaceAll(s, "<SP>", " ")))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mustValid(t *testing.T, d *Doc) map[string]any {
	t.Helper()
	std, ok := d.Get("")
	if !ok {
		t.Fatal("root missing")
	}
	var v map[string]any
	if err := json.Unmarshal(std, &v); err != nil {
		t.Fatalf("result is not valid JSON: %v\n%s", err, d.Bytes())
	}
	return v
}

func TestSetKeepsCommentsAndIndentation(t *testing.T) {
	d := mustParse(t, terminalSettings)
	if err := d.Set("/profiles/defaults/colorScheme", []byte(`"Aether"`)); err != nil {
		t.Fatal(err)
	}
	if err := d.Append("/schemes", []byte(`{"name":"Aether","background":"#000000"}`)); err != nil {
		t.Fatal(err)
	}
	out := string(d.Bytes())
	for _, want := range []string{
		"// keep this comment",
		"// trailing comment",
		"    \"keybindings\": \n    [",
		"        \"defaults\": {\n            \"colorScheme\": \"Aether\"\n        },",
		"    \"schemes\": [\n        {\n            \"name\": \"Aether\",\n            \"background\": \"#000000\"\n        }\n    ]\n}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	v := mustValid(t, d)
	if got := v["profiles"].(map[string]any)["defaults"].(map[string]any)["colorScheme"]; got != "Aether" {
		t.Errorf("colorScheme = %v", got)
	}
}

func TestSetReplacesExistingValueInPlace(t *testing.T) {
	d := mustParse(t, terminalSettings)
	i := d.FindBy("/profiles/list", "guid", "{574e775e-4f2a-5b96-ac1e-a2962a402336}")
	if i != 0 {
		t.Fatalf("FindBy = %d", i)
	}
	if err := d.Set("/profiles/list/0/name", []byte(`"pwsh"`)); err != nil {
		t.Fatal(err)
	}
	out := string(d.Bytes())
	if !strings.Contains(out, `"name": "pwsh", // trailing comment`) {
		t.Errorf("replacement lost its trailing comment:\n%s", out)
	}
	mustValid(t, d)
}

func TestAppendAfterTrailingCommentKeepsComment(t *testing.T) {
	d := mustParse(t, "{\n  \"a\": 1 // one\n}\n")
	if err := d.Set("/b", []byte(`2`)); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"a\": 1, // one\n  \"b\": 2\n}\n"
	if got := string(d.Bytes()); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineObjectsStayInline(t *testing.T) {
	d := mustParse(t, "{\n  \"rules\": [\n    { \"kind\": \"Exe\", \"id\": \"a.exe\" }\n  ]\n}\n")
	if err := d.Set("/rules/0/matching_strategy", []byte(`"Equals"`)); err != nil {
		t.Fatal(err)
	}
	want := `{ "kind": "Exe", "id": "a.exe", "matching_strategy": "Equals" }`
	if !strings.Contains(string(d.Bytes()), want) {
		t.Errorf("inline object was expanded:\n%s", d.Bytes())
	}
}

func TestMergePatch(t *testing.T) {
	d := mustParse(t, `{
  "border": true,
  "theme": {
    "palette": "Base16",
    "name": "Ashes",
    "unfocused_border": "Base03"
  },
  "monitors": []
}
`)
	patch := `{
  // comments are allowed in patches
  "theme": {
    "palette": "Custom",
    "name": null,
    "colours": {"base_00": "#101010", "base_01": "#202020"}
  },
  "new": [1, 2]
}`
	if err := d.MergePatch("", []byte(patch)); err != nil {
		t.Fatal(err)
	}
	v := mustValid(t, d)
	theme := v["theme"].(map[string]any)
	if theme["palette"] != "Custom" || theme["unfocused_border"] != "Base03" {
		t.Errorf("theme = %v", theme)
	}
	if _, ok := theme["name"]; ok {
		t.Error("name was not removed")
	}
	if theme["colours"].(map[string]any)["base_01"] != "#202020" {
		t.Errorf("colours = %v", theme["colours"])
	}
	if v["border"] != true || v["monitors"] == nil || v["new"] == nil {
		t.Errorf("document = %v", v)
	}
	if !strings.Contains(string(d.Bytes()), "\n    \"colours\": {\n      \"base_00\": \"#101010\",\n") {
		t.Errorf("new member not indented like its siblings:\n%s", d.Bytes())
	}
}

func TestDeleteAndFindByName(t *testing.T) {
	d := mustParse(t, `{"schemes": [{"name": "A"}, {"name": "Aether"}, {"name": "B"}]}`)
	i := d.FindByName("/schemes", "Aether")
	if i != 1 {
		t.Fatalf("FindByName = %d", i)
	}
	if err := d.Delete("/schemes/1"); err != nil {
		t.Fatal(err)
	}
	if d.FindByName("/schemes", "Aether") != -1 || d.Len("/schemes") != 2 {
		t.Errorf("delete failed: %s", d.Bytes())
	}
	if err := d.Delete("/missing/x"); err != nil {
		t.Errorf("delete of a missing pointer: %v", err)
	}
}

func TestParseRejectsNonObjects(t *testing.T) {
	if _, err := Parse([]byte(`[1]`)); err == nil {
		t.Error("array root accepted")
	}
	if _, err := Parse([]byte(`{"a": `)); err == nil {
		t.Error("truncated document accepted")
	}
	d, err := Parse([]byte("\xef\xbb\xbf{\"a\": 1}"))
	if err != nil {
		t.Fatalf("BOM: %v", err)
	}
	if s, ok := d.Get("/a"); !ok || string(s) != "1" {
		t.Errorf("Get after BOM = %s", s)
	}
}

func TestEnsureRejectsWrongKind(t *testing.T) {
	d := mustParse(t, `{"profiles": []}`)
	if err := d.Ensure("/profiles", "{}"); err == nil {
		t.Error("Ensure accepted an array where an object was required")
	}
	if err := d.Ensure("/schemes", "[]"); err != nil || d.Kind("/schemes") != '[' {
		t.Errorf("Ensure array: %v %c", err, d.Kind("/schemes"))
	}
}
