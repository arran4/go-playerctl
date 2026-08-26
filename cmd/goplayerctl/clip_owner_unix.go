//go:build !windows && !darwin

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.design/x/clipboard"
)

const internalClipboardOwnerArg = "--internal-clipboard-owner"

type clipboardOwnerProcess interface {
	Payload() io.WriteCloser
	Status() io.ReadCloser
	Wait() error
	Release() error
}

type execClipboardOwnerProcess struct {
	cmd     *exec.Cmd
	payload *os.File
	status  *os.File
}

func (p *execClipboardOwnerProcess) Payload() io.WriteCloser { return p.payload }
func (p *execClipboardOwnerProcess) Status() io.ReadCloser   { return p.status }
func (p *execClipboardOwnerProcess) Wait() error             { return p.cmd.Wait() }
func (p *execClipboardOwnerProcess) Release() error          { return p.cmd.Process.Release() }

var startClipboardOwner = startExecClipboardOwner

func startExecClipboardOwner() (clipboardOwnerProcess, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("find executable: %w", err)
	}

	payloadReader, payloadWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create payload pipe: %w", err)
	}
	statusReader, statusWriter, err := os.Pipe()
	if err != nil {
		_ = payloadReader.Close()
		_ = payloadWriter.Close()
		return nil, fmt.Errorf("create status pipe: %w", err)
	}

	cmd := exec.Command(executable, internalClipboardOwnerArg)
	cmd.Stdin = payloadReader
	cmd.ExtraFiles = []*os.File{statusWriter}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = payloadReader.Close()
		_ = payloadWriter.Close()
		_ = statusReader.Close()
		_ = statusWriter.Close()
		return nil, fmt.Errorf("start clipboard owner: %w", err)
	}
	_ = payloadReader.Close()
	_ = statusWriter.Close()
	return &execClipboardOwnerProcess{cmd: cmd, payload: payloadWriter, status: statusReader}, nil
}

type unixClipboardCopier struct{}

func newPlatformClipboardCopier() clipboardCopier { return unixClipboardCopier{} }

func (unixClipboardCopier) Copy(text string) error {
	process, err := startClipboardOwner()
	if err != nil {
		return err
	}

	failed := func(cause error) error {
		_ = process.Payload().Close()
		_ = process.Status().Close()
		waitErr := process.Wait()
		if waitErr != nil && cause == nil {
			return waitErr
		}
		return cause
	}

	if _, err := io.WriteString(process.Payload(), text); err != nil {
		return failed(fmt.Errorf("send clipboard payload: %w", err))
	}
	if err := process.Payload().Close(); err != nil {
		return failed(fmt.Errorf("close clipboard payload: %w", err))
	}

	status, err := bufio.NewReader(process.Status()).ReadString('\n')
	_ = process.Status().Close()
	if err != nil {
		return failed(fmt.Errorf("clipboard owner exited before READY: %w", err))
	}
	status = strings.TrimSpace(status)
	switch {
	case status == "READY":
		if err := process.Release(); err != nil {
			return fmt.Errorf("release clipboard owner: %w", err)
		}
		return nil
	case strings.HasPrefix(status, "ERROR:"):
		return failed(errors.New(strings.TrimSpace(strings.TrimPrefix(status, "ERROR:"))))
	default:
		return failed(fmt.Errorf("unexpected clipboard owner status %q", status))
	}
}

var (
	clipboardInit  = clipboard.Init
	clipboardWrite = func(payload []byte) (<-chan struct{}, error) {
		return clipboard.Write(context.Background(), clipboard.FmtText, payload)
	}
)

func handleInternalClipboardOwner(args []string) (bool, int) {
	if len(args) == 1 && args[0] == internalClipboardOwnerArg {
		return true, runClipboardOwner(os.Stdin, os.NewFile(3, "clipboard-status"))
	}
	return false, 0
}

func runClipboardOwner(payloadReader io.Reader, status io.WriteCloser) int {
	if status == nil {
		fmt.Fprintln(os.Stderr, "internal error: clipboard status pipe not provided")
		return 1
	}
	signal := func(message string) {
		_, _ = io.WriteString(status, message+"\n")
		_ = status.Close()
	}
	payload, err := io.ReadAll(payloadReader)
	if err != nil {
		signal("ERROR: read payload: " + err.Error())
		return 1
	}
	if err := clipboardInit(); err != nil {
		signal("ERROR: initialize clipboard: " + err.Error())
		return 1
	}
	changed, err := clipboardWrite(payload)
	if err != nil {
		signal("ERROR: write clipboard: " + err.Error())
		return 1
	}
	signal("READY")
	if changed != nil {
		<-changed
	}
	return 0
}
