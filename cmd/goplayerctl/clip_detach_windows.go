//go:build windows || darwin

package main

import (
	"golang.design/x/clipboard"
)

type nativeClipboardCopier struct{}

func (n *nativeClipboardCopier) Copy(text string) error {
	if err := clipboard.Init(); err != nil {
		return err
	}
	changed := clipboard.Write(clipboard.FmtText, []byte(text))
	// On Windows and macOS, the system clipboard holds the data.
	// We don't need a persistent process.
	// We just drop the channel.
	_ = changed
	return nil
}

// We don't need getSysProcAttr here anymore, but keeping it empty if we wanted.
// Wait, we don't need the internal daemon on windows either.
