//go:build darwin

package main

import (
	"context"

	"golang.design/x/clipboard"
)

type directClipboardCopier struct{}

func newPlatformClipboardCopier() clipboardCopier { return directClipboardCopier{} }

func (directClipboardCopier) Copy(text string) error {
	if err := clipboard.Init(); err != nil {
		return err
	}
	_, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(text))
	return err
}

func handleInternalClipboardOwner(args []string) (bool, int) { return false, 0 }
