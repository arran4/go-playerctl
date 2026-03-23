package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/arran4/go-playerctl/pkg/playerctl"
)

func TestDaemonVersionAndInitFailure(t *testing.T) {
	orig := newPlayerManger
	defer func() { newPlayerManger = orig }()

	var out, errOut bytes.Buffer
	if code := runDaemon([]string{"--version"}, &out, &errOut); code != 0 {
		t.Fatalf("version failed: code=%d", code)
	}
	out.Reset()
	errOut.Reset()
	if code := runDaemon([]string{"-v"}, &out, &errOut); code != 0 {
		t.Fatalf("-v failed: code=%d", code)
	}

	newPlayerManger = func(source playerctl.Source) (*playerctl.PlayerManager, error) { return nil, errors.New("boom") }
	out.Reset()
	errOut.Reset()
	if code := runDaemon([]string{"--once"}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "daemon init failed") {
		t.Fatalf("expected init failure code=1: got %d err=%q", code, errOut.String())
	}
}

func TestDaemonShiftUnshift(t *testing.T) {
	d := &daemon{players: []string{"a", "b", "c"}}
	_ = d.Shift()
	n, _ := d.PlayerNames()
	if strings.Join(n, ",") != "b,c,a" {
		t.Fatalf("shift got %v", n)
	}
	_ = d.Unshift()
	n, _ = d.PlayerNames()
	if strings.Join(n, ",") != "a,b,c" {
		t.Fatalf("unshift got %v", n)
	}
}

func TestDaemonActivePlayerTracking(t *testing.T) {
	d := &daemon{players: []string{"x", "y"}, active: "x", lastActivityUnix: 1}
	_ = d.Shift()
	active, ts, _ := d.ActivePlayer()
	if active != "y" || ts <= 1 {
		t.Fatalf("unexpected active tracking: %s %d", active, ts)
	}
}

func TestDaemonConcurrentShiftUnshift(t *testing.T) {
	d := &daemon{players: []string{"a", "b", "c", "d"}, active: "a", lastActivityUnix: 1}
	done := make(chan struct{})
	go func() {
		for i := 0; i < 200; i++ {
			_ = d.Shift()
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < 200; i++ {
			_ = d.Unshift()
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	n, _ := d.PlayerNames()
	if len(n) != 4 {
		t.Fatalf("unexpected player list size after concurrent operations: %d", len(n))
	}
}
