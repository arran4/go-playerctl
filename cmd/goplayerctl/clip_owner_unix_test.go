//go:build !windows && !darwin

package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type bufferWriteCloser struct{ bytes.Buffer }

func (b *bufferWriteCloser) Close() error { return nil }

type fakeOwnerProcess struct {
	payload  *bufferWriteCloser
	status   io.ReadCloser
	waited   bool
	released bool
	waitErr  error
}

func (p *fakeOwnerProcess) Payload() io.WriteCloser { return p.payload }
func (p *fakeOwnerProcess) Status() io.ReadCloser   { return p.status }
func (p *fakeOwnerProcess) Wait() error             { p.waited = true; return p.waitErr }
func (p *fakeOwnerProcess) Release() error          { p.released = true; return nil }

func ownerProcessWithStatus(status string) *fakeOwnerProcess {
	return &fakeOwnerProcess{payload: &bufferWriteCloser{}, status: io.NopCloser(strings.NewReader(status))}
}

func withOwnerProcess(t *testing.T, process *fakeOwnerProcess) {
	t.Helper()
	original := startClipboardOwner
	startClipboardOwner = func() (clipboardOwnerProcess, error) { return process, nil }
	t.Cleanup(func() { startClipboardOwner = original })
}

func TestUnixCopierReadyTransfersExactPayloadAndReleases(t *testing.T) {
	process := ownerProcessWithStatus("READY\n")
	withOwnerProcess(t, process)
	payload := "Björk — 世界\n"
	if err := (unixClipboardCopier{}).Copy(payload); err != nil {
		t.Fatal(err)
	}
	if process.payload.String() != payload || !process.released || process.waited {
		t.Fatalf("payload=%q released=%v waited=%v", process.payload.String(), process.released, process.waited)
	}
}

func TestUnixCopierFailuresAreReaped(t *testing.T) {
	for _, status := range []string{"ERROR: display unavailable\n", "", "WHAT\n"} {
		t.Run(status, func(t *testing.T) {
			process := ownerProcessWithStatus(status)
			withOwnerProcess(t, process)
			if err := (unixClipboardCopier{}).Copy("value"); err == nil {
				t.Fatal("expected protocol failure")
			}
			if !process.waited || process.released {
				t.Fatalf("waited=%v released=%v", process.waited, process.released)
			}
		})
	}
}

func TestUnixCopierWaitsForReady(t *testing.T) {
	statusReader, statusWriter := io.Pipe()
	process := &fakeOwnerProcess{payload: &bufferWriteCloser{}, status: statusReader}
	withOwnerProcess(t, process)
	done := make(chan error, 1)
	go func() { done <- (unixClipboardCopier{}).Copy("payload") }()
	select {
	case err := <-done:
		t.Fatalf("copy returned before READY: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	_, _ = io.WriteString(statusWriter, "READY\n")
	_ = statusWriter.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestClipboardOwnerSignalsReadyAndPreservesUnicode(t *testing.T) {
	originalInit, originalWrite := clipboardInit, clipboardWrite
	clipboardInit = func() error { return nil }
	var got []byte
	changed := make(chan struct{})
	close(changed)
	clipboardWrite = func(payload []byte) (<-chan struct{}, error) {
		got = append([]byte(nil), payload...)
		return changed, nil
	}
	t.Cleanup(func() { clipboardInit, clipboardWrite = originalInit, originalWrite })
	status := &bufferWriteCloser{}
	if code := runClipboardOwner(strings.NewReader("世界 🎵"), status); code != 0 {
		t.Fatalf("code=%d status=%q", code, status.String())
	}
	if string(got) != "世界 🎵" || status.String() != "READY\n" {
		t.Fatalf("payload=%q status=%q", got, status.String())
	}
}

func TestClipboardOwnerSignalsInitAndWriteErrors(t *testing.T) {
	for _, writeFails := range []bool{false, true} {
		originalInit, originalWrite := clipboardInit, clipboardWrite
		clipboardInit = func() error {
			if !writeFails {
				return errors.New("init failed")
			}
			return nil
		}
		clipboardWrite = func([]byte) (<-chan struct{}, error) { return nil, errors.New("write failed") }
		status := &bufferWriteCloser{}
		if code := runClipboardOwner(strings.NewReader("x"), status); code == 0 || !strings.HasPrefix(status.String(), "ERROR:") {
			t.Fatalf("code=%d status=%q", code, status.String())
		}
		clipboardInit, clipboardWrite = originalInit, originalWrite
	}
}
