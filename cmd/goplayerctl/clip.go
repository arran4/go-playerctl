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
