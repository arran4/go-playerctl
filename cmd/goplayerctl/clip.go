package main

type clipboardCopier interface {
	Copy(text string) error
}

var defaultClipboardCopier clipboardCopier

func copyToClipboard(text string) error {
	if defaultClipboardCopier != nil {
		return defaultClipboardCopier.Copy(text)
	}
	return nil
}
