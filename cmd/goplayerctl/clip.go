package main

import "errors"

type clipboardCopier interface {
	Copy(text string) error
}

var defaultClipboardCopier clipboardCopier = newPlatformClipboardCopier()

func copyToClipboard(text string) error {
	if defaultClipboardCopier == nil {
		return errors.New("clipboard copier is not initialized")
	}
	return defaultClipboardCopier.Copy(text)
}
