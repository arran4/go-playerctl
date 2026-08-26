package main

import "errors"

type clipboardCopier interface {
	Copy(text string) error
}

var defaultClipboardCopier clipboardCopier

func copyToClipboard(text string) error {
	if defaultClipboardCopier != nil {
		return defaultClipboardCopier.Copy(text)
	}
	return errors.New("clipboard copying is not implemented or initialized on this platform")
}

// handleInternalClipboardOwner handles the hidden `--internal-clipboard-owner` argument.
// It returns (true, exitCode) if it was handled, or (false, 0) otherwise.
func handleInternalClipboardOwner(args []string) (bool, int) {
	if len(args) > 0 && args[0] == "--internal-clipboard-owner" {
		return true, runClipboardOwner()
	}
	return false, 0
}
