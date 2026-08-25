//go:build !windows

package main

import "syscall"

func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true, // run in a new session, detach from controlling terminal
	}
}
