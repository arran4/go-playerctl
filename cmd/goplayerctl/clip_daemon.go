package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.design/x/clipboard"
)

func runClipboardOwner() int {
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

	_ = os.Stdin.Close()
	_ = os.Stdout.Close()
	_ = os.Stderr.Close()

	if changed != nil {
		<-changed
	}

	return 0
}
