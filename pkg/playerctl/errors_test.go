package playerctl

import "testing"

func TestInvalidCommandError(t *testing.T) {
	tests := []struct {
		name    string
		err     InvalidCommandError
		expects string
	}{
		{name: "with command", err: InvalidCommandError{Command: "dance"}, expects: "invalid command: dance"},
		{name: "without command", err: InvalidCommandError{}, expects: "invalid command"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expects {
				t.Fatalf("InvalidCommandError.Error() = %q, want %q", got, tt.expects)
			}
		})
	}
}

func TestFormatError(t *testing.T) {
	tests := []struct {
		name    string
		err     FormatError
		expects string
	}{
		{name: "with message", err: FormatError{Message: "expected }}"}, expects: "format error: expected }}"},
		{name: "without message", err: FormatError{}, expects: "format error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expects {
				t.Fatalf("FormatError.Error() = %q, want %q", got, tt.expects)
			}
		})
	}
}

func TestErrPlayerNotFound(t *testing.T) {
	if got := ErrPlayerNotFound.Error(); got != "player not found" {
		t.Fatalf("ErrPlayerNotFound.Error() = %q, want %q", got, "player not found")
	}
}
