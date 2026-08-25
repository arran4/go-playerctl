package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type clipboardCopier interface {
	Copy(text string) error
}

type nativeClipboardCopier struct{}

func (n *nativeClipboardCopier) Copy(text string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not find executable: %w", err)
	}

	cmd := exec.Command(exe, "--internal-clipboard-owner")

	// Set up pipes
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create status pipe: %w", err)
	}
	defer r.Close()

	cmd.ExtraFiles = []*os.File{w}
	cmd.Stdin = strings.NewReader(text)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = getSysProcAttr()

	if err := cmd.Start(); err != nil {
		w.Close()
		return fmt.Errorf("failed to start clipboard owner: %w", err)
	}

	w.Close() // Close writer in parent so reader gets EOF when child exits or closes it

	// Wait for READY or ERROR handshake
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		status := scanner.Text()
		if status == "READY" {
			// Detach from the child process.
			// Releasing resources and ignoring the exit state since it's meant to outlive us.
			_ = cmd.Process.Release()
			return nil
		}
		if strings.HasPrefix(status, "ERROR: ") {
			// Wait for the process to exit so it's fully cleaned up since it failed
			_ = cmd.Wait()
			return errors.New(strings.TrimPrefix(status, "ERROR: "))
		}
		_ = cmd.Wait()
		return fmt.Errorf("unexpected status from clipboard owner: %q", status)
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		return fmt.Errorf("failed to read status from clipboard owner: %w", err)
	}

	_ = cmd.Wait()
	return errors.New("clipboard owner exited prematurely without status")
}

var defaultClipboardCopier clipboardCopier = &nativeClipboardCopier{}

func copyToClipboard(text string) error {
	return defaultClipboardCopier.Copy(text)
}
