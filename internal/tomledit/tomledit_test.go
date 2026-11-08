package tomledit

import (
	"strings"
	"testing"
)

const leopard = `# LeopardWM config with comments
[layout]
gap = 10
width_presets = [
    0.333, # one third
    0.5,
]

[appearance]
language = "en" # interface language
active_border_color = '4285F4'

[hotkeys]
scroll_modifier = "Win"
"Win+H" = "focus_left"
"Win+[" = "consume_window" # brackets inside quoted keys
note = """
[appearance]
not a header
"""

[[window_rules]]
match_executable = "League of Legends.exe"
action = "ignore"
`

func TestEditKeepsEverythingElse(t *testing.T) {
	out, err := Edit([]byte(leopard), "appearance", []Change{
		{Key: "active_border_color", Value: String("A1B2C3")},
		{Key: "tab_strip_bg", Value: String("101010")},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(leopard,
		"active_border_color = '4285F4'\n",
		"active_border_color = \"A1B2C3\"\ntab_strip_bg = \"101010\"\n", 1)
	if string(out) != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestEditCreatesMissingTable(t *testing.T) {
	src := "[layout]\r\ngap = 10\r\n\r\n[[window_rules]]\r\naction = \"float\"\r\n"
	out, err := Edit([]byte(src), "appearance", []Change{{Key: "tab_strip_bg", Value: String("101010")}})
	if err != nil {
		t.Fatal(err)
	}
	want := src + "\r\n[appearance]\r\ntab_strip_bg = \"101010\"\r\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", out, want)
	}
	got, ok, err := Get(out, "appearance", "tab_strip_bg")
	if err != nil || !ok || got != `"101010"` {
		t.Errorf("Get = %q %v %v", got, ok, err)
	}
}

func TestEditDeletesAndRestores(t *testing.T) {
	out, err := Edit([]byte(leopard), "appearance", []Change{{Key: "active_border_color", Delete: true}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "active_border_color") {
		t.Errorf("key not deleted:\n%s", out)
	}
	if _, ok, _ := Get(out, "appearance", "language"); !ok {
		t.Error("neighbouring key lost")
	}
}

func TestEditEmptyTableAndNoTrailingNewline(t *testing.T) {
	out, err := Edit([]byte("[appearance]"), "appearance", []Change{{Key: "a", Value: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "[appearance]\na = 1\n" {
		t.Errorf("got %q", out)
	}
}

func TestEditRefusesUnsupportedLayouts(t *testing.T) {
	for name, src := range map[string]string{
		"invalid toml": "[appearance\n",
		"dotted keys":  "appearance.tab_strip_bg = \"000000\"\n",
		"inline table": "appearance = { tab_strip_bg = \"000000\" }\n",
	} {
		if _, err := Edit([]byte(src), "appearance", []Change{{Key: "tab_strip_bg", Value: String("101010")}}); err == nil {
			t.Errorf("%s: edit accepted", name)
		}
	}
}

func TestNormalizeKey(t *testing.T) {
	for raw, want := range map[string]string{
		`appearance`:        "appearance",
		` a . "b c" `:       "a.b c",
		`'lit.eral'`:        "lit.eral",
		`"Win+H"`:           "Win+H",
		`"quote \" inside"`: `quote " inside`,
		`site."google.com"`: "site.google.com",
		`bare-key_1`:        "bare-key_1",
	} {
		if got := normalizeKey(raw); got != want {
			t.Errorf("normalizeKey(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestStringEscapes(t *testing.T) {
	if got := String("a\"b\\c\n"); got != `"a\"b\\c\n"` {
		t.Errorf("String = %s", got)
	}
}
