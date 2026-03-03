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
