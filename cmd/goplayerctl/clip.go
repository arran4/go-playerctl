package main

import (
    "context"
	"golang.design/x/clipboard"
)

type clipboardWriter interface {
	Init() error
	WriteText(text string) error
}

type nativeClipboardWriter struct{
    initialized bool
}

func (w *nativeClipboardWriter) Init() error {
    if w.initialized {
        return nil
    }
	err := clipboard.Init()
    if err == nil {
        w.initialized = true
    }
    return err
}

func (w *nativeClipboardWriter) WriteText(text string) error {
	_, _ = clipboard.Write(context.Background(), clipboard.FmtText, []byte(text))
	return nil
}

var defaultClipboardWriter clipboardWriter = &nativeClipboardWriter{}

func copyToClipboard(text string) error {
	if err := defaultClipboardWriter.Init(); err != nil {
		return err
	}
	return defaultClipboardWriter.WriteText(text)
}
