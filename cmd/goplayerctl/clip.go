package main

import (
	"errors"
	"os/exec"
	"strings"
)

// clipboardExecutor defines how commands are located and executed
type clipboardExecutor interface {
	LookPath(file string) (string, error)
	Run(name string, stdin string, args ...string) error
}

// realClipboardExecutor is the default implementation that talks to the OS
type realClipboardExecutor struct{}

func (e *realClipboardExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (e *realClipboardExecutor) Run(name string, stdin string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	return cmd.Run()
}

// defaultClipboardExec is used by default but can be swapped out for tests
var defaultClipboardExec clipboardExecutor = &realClipboardExecutor{}

// copyToClipboard tries various system clipboard utilities to copy the given text.
func copyToClipboard(text string) error {
	commands := []struct {
		name string
		args []string
	}{
		{"wl-copy", []string{}},
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
		{"pbcopy", []string{}}, // macOS
		{"clip.exe", []string{}}, // Windows
	}

	var lastErr error
	foundCommand := false
	for _, cmdConfig := range commands {
		if _, err := defaultClipboardExec.LookPath(cmdConfig.name); err == nil {
			foundCommand = true
			if err := defaultClipboardExec.Run(cmdConfig.name, text, cmdConfig.args...); err != nil {
				lastErr = err
				continue
			}
			return nil
		}
	}

	if !foundCommand {
		return errors.New("no supported clipboard utility found (wl-copy, xclip, xsel, pbcopy, clip.exe)")
	}
	return lastErr
}
