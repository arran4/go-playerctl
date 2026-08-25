package main

import (
	"errors"
	"os/exec"
	"strings"
)

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
	for _, cmdConfig := range commands {
		if _, err := exec.LookPath(cmdConfig.name); err == nil {
			cmd := exec.Command(cmdConfig.name, cmdConfig.args...)
			cmd.Stdin = strings.NewReader(text)
			if err := cmd.Run(); err != nil {
				lastErr = err
				continue
			}
			return nil
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("no supported clipboard utility found (wl-copy, xclip, xsel, pbcopy, clip.exe)")
}
