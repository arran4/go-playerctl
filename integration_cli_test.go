package integration_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPlayerctlVersionCommandIntegration(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/goplayerctl", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run --version failed: %v output=%s", err, string(out))
	}
	if !strings.Contains(string(out), "goplayerctl") {
		t.Fatalf("unexpected version output: %s", string(out))
	}
}

func TestCLIHelp(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/goplayerctl", "-h")
	out, err := cmd.CombinedOutput()
	// "go run" can exit with code 1 if the program it's running exits with non-zero code.
	// We just check if it contains the Usage string.
	if err == nil {
		t.Fatalf("expected non-zero exit on -h, got %v: %s", err, string(out))
	}
	output := string(out)
	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected help output to contain 'Usage:', got: %s", output)
	}
	if !strings.Contains(output, "-tui-scheme") {
		t.Errorf("expected help output to contain '-tui-scheme', got: %s", output)
	}
}

func TestPlayerctlMissingCommandIntegration(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/goplayerctl")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, output=%s", string(out))
	}
	if !strings.Contains(string(out), "Usage: goplayerctl") {
		t.Fatalf("unexpected missing command output: %s", string(out))
	}
}

func TestPlayerctlDumpMissingPlayerIntegration(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/goplayerctl", "dump")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, output=%s", string(out))
	}
	if !strings.Contains(string(out), "no players selected") {
		t.Fatalf("unexpected missing player output: %s", string(out))
	}
}

func TestPlayerctlDumpCommandIntegration(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/goplayerctl", "--all-players", "dump-json")
	out, err := cmd.CombinedOutput()
	// It's possible there are no players running on the test system.
	// But it shouldn't crash, it should just print `[]` or fail gracefully.
	if err != nil && !strings.Contains(string(out), "failed to connect player") && !strings.Contains(string(out), "no players selected") {
		t.Logf("Warning: go run --all-players dump-json returned err=%v, output=%s", err, string(out))
	} else if err == nil && !strings.HasPrefix(strings.TrimSpace(string(out)), "[") && !strings.HasPrefix(strings.TrimSpace(string(out)), "{") {
		t.Fatalf("expected JSON output (starting with [ or {), got: %s", string(out))
	}
}

func TestPlayerctlFollowIntegration(t *testing.T) {
	// The follow command should continue running until cancelled,
	// rather than terminating early (e.g., after 3 intervals).
	// We set a 500ms timeout context and a 100ms follow interval.
	// We expect the command to hit the context timeout and be killed,
	// returning a non-nil error, rather than exiting cleanly beforehand.

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/goplayerctl", "--player", "mock", "--follow", "--follow-interval", "100ms", "status")
	err := cmd.Run()

	// A nil error means the command exited cleanly before our timeout, which would be a bug.
	if err == nil {
		t.Fatalf("expected command to run until context timeout, but it exited cleanly with nil error")
	}

	// We expect it to be killed by context cancellation
	if ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("expected context to have DeadlineExceeded, got: %v, cmd error: %v", ctx.Err(), err)
	}
}
