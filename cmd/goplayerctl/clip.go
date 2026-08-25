package main

type clipboardCopier interface {
	Copy(text string) error
}

var defaultClipboardCopier clipboardCopier = &nativeClipboardCopier{}

func copyToClipboard(text string) error {
	return defaultClipboardCopier.Copy(text)
}
