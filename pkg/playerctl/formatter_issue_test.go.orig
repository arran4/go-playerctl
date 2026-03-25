package playerctl

import "testing"

func TestFormatterIssue(t *testing.T) {
	f, err := NewFormatter("{{ .artist }}")
	if err != nil {
		t.Fatal(err)
	}
	ctx := map[string]any{
		"artist": "Test",
		"xesam:album": "Album",
	}
	_, err = f.Expand(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
