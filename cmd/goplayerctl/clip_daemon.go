//go:build !windows && !darwin

package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.design/x/clipboard"
)

// runClipboardOwner is the entry point for the internal clipboard daemon.
// It reads the payload from stdin, initializes the clipboard, writes it,
// sends "READY" or "ERROR" to the status pipe (fd 3), and then blocks until overwritten.
func handleInternalClipboardOwner() int {
	// Status pipe is passed as ExtraFiles[0], which is fd 3
	statusFile := os.NewFile(3, "status_pipe")
	if statusFile == nil {
		fmt.Fprintln(os.Stderr, "internal error: status pipe fd 3 not provided")
		return 1
	}

	sendStatus := func(msg string) {
		_, _ = statusFile.Write([]byte(msg + "\n"))
		_ = statusFile.Close()
	}

	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		sendStatus("ERROR: failed to read payload: " + err.Error())
		return 1
	}

	if err := clipboard.Init(); err != nil {
		sendStatus("ERROR: clipboard init failed: " + err.Error())
		return 1
	}

	changed, err := clipboard.Write(context.Background(), clipboard.FmtText, payload)
	if err != nil {
		sendStatus("ERROR: clipboard write failed: " + err.Error())
		return 1
	}

	sendStatus("READY")

	// Detach completely by closing stdio
	_ = os.Stdin.Close()
	_ = os.Stdout.Close()
	_ = os.Stderr.Close()

	if changed != nil {
		<-changed
	}

	return 0
}
