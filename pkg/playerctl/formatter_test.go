package playerctl

import "testing"

func TestNewFormatter(t *testing.T) {
	if _, err := NewFormatter("  "); err == nil {
		t.Fatal("expected error for empty template")
	}

	if _, err := NewFormatter("{{ .title }}"); err != nil {
		t.Fatalf("unexpected error creating formatter: %v", err)
	}
}

func TestFormatterContainsKey(t *testing.T) {
	f, err := NewFormatter("{{ .artist }} - {{ .title }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !f.ContainsKey("artist") || !f.ContainsKey("title") {
		t.Fatal("expected artist and title keys")
	}
	if f.ContainsKey("album") {
		t.Fatal("did not expect album key to be present")
	}
}

func TestFormatterExpand(t *testing.T) {
	f, err := NewFormatter("{{ .artist }} - {{ .title }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := f.Expand(map[string]any{"artist": "Boards of Canada", "title": "Dayvan Cowboy"})
	if err != nil || got != "Boards of Canada - Dayvan Cowboy" {
		t.Fatalf("Expand() = %q, err=%v", got, err)
	}

	got, err = f.Expand(map[string]any{"artist": "Tycho"})
	if err != nil || got != "Tycho - <no value>" {
		t.Fatalf("Expand() missing key = %q, err=%v", got, err)
	}
}

func TestFormatterExpandFuncsAndParseError(t *testing.T) {
	f, err := NewFormatter(`{{ uc .artist }} {{ default .title "Untitled" }}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := f.Expand(map[string]any{"artist": "nils"})
	if err != nil || got != "NILS Untitled" {
		t.Fatalf("Expand() funcs = %q, err=%v", got, err)
	}

	if _, err := NewFormatter("{{ .artist "); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestFormatterArithmetic(t *testing.T) {
	f, err := NewFormatter(`{{ add 2 3 }} {{ sub 9 4 }}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := f.Expand(map[string]any{})
	if err != nil || got != "5 5" {
		t.Fatalf("arithmetic mismatch got=%q err=%v", got, err)
	}
}

func TestFormatterHelperParityFunctions(t *testing.T) {
	f, err := NewFormatter(`{{ duration .d }}|{{ markup_escape .m }}|{{ emoji .s }}|{{ trunc .t 5 }}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := f.Expand(map[string]any{
		"d": "125s",
		"m": `<b>&</b>`,
		"s": "playing",
		"t": "abcdefgh",
	})
	if err != nil {
		t.Fatalf("unexpected expand error: %v", err)
	}
	if got != "2:05|&lt;b&gt;&amp;&lt;/b&gt;|▶️|abcd…" {
		t.Fatalf("helper parity mismatch: %q", got)
	}
}

func TestFormatterExpandBareWords(t *testing.T) {
	f, err := NewFormatter("{{ artist }} - {{ default title \"Untitled\" }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := f.Expand(map[string]any{"artist": "Boards of Canada", "title": "Dayvan Cowboy"})
	if err != nil || got != "Boards of Canada - Dayvan Cowboy" {
		t.Fatalf("Expand bare words = %q, err=%v", got, err)
	}
}

func TestFormatterExpandBareWordsMissing(t *testing.T) {
	f, err := NewFormatter("{{ artist }} - {{ default title \"Untitled\" }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := f.Expand(map[string]any{"artist": "Boards of Canada"})
	if err != nil || got != "Boards of Canada - Untitled" {
		t.Fatalf("Expand missing bare words = %q, err=%v", got, err)
	}
}

func TestFormatterBuiltinFunctions(t *testing.T) {
	// Verify that the built-in functions still work and aren't overwritten
	f, err := NewFormatter("{{ if eq .status \"Playing\" }}Yes{{ end }} - {{ len .title }} - {{ print .artist }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := f.Expand(map[string]any{"status": "Playing", "title": "12345", "artist": "Boards"})
	if err != nil || got != "Yes - 5 - Boards" {
		t.Fatalf("Expand built-ins = %q, err=%v", got, err)
	}
}
