package integration_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPlayerctlVersionCommandIntegration(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/goplayerctl", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run --version failed: %v output=%s", err, string(out))
	}
	if !strings.Contains(string(out), "go-playerctl") {
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
	if !strings.Contains(output, "Usage of") {
		t.Errorf("expected help output to contain 'Usage of', got: %s", output)
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
	if !strings.Contains(string(out), "missing command") {
		t.Fatalf("unexpected missing command output: %s", string(out))
	}
}
