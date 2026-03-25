package playerctl

import "testing"

func TestFormatterIssue(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		context map[string]any
		want    string
	}{
		{
			name:   "panic on invalid identifiers",
			format: "{{ .artist }}",
			context: map[string]any{
				"artist":      "Test",
				"xesam:album": "Album",
			},
			want: "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFormatter(tt.format)
			if err != nil {
				t.Fatalf("unexpected error creating formatter: %v", err)
			}

			got, err := f.Expand(tt.context)
			if err != nil {
				t.Fatalf("unexpected error expanding template: %v", err)
			}

			if got != tt.want {
				t.Errorf("Expand() = %v, want %v", got, tt.want)
			}
		})
	}
}
