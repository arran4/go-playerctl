package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/arran4/go-playerctl/pkg/playerctl"
)

func TestRunValidationAndVersion(t *testing.T) {
	var out, errOut bytes.Buffer

	code := run([]string{"--version"}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), "go-playerctl") {
		t.Fatalf("version failed: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"status"}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "no players selected") {
		t.Fatalf("missing player check failed: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}

	orig := newPlayer
	origManager := newPlayerManger
	defer func() { newPlayer = orig; newPlayerManger = origManager }()
	newPlayer = func(instance string, source playerctl.Source) (*playerctl.Player, error) {
		return &playerctl.Player{}, nil
	}
	newPlayerManger = func(source playerctl.Source) (*playerctl.PlayerManager, error) {
		return &playerctl.PlayerManager{}, nil
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"--player", "vlc", "nope"}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("unknown command check failed: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"--template-help"}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), "Format strings use standard Go `text/template` syntax.") {
		t.Fatalf("template help failed: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestRunConnectionFailure(t *testing.T) {
	orig := newPlayer
	defer func() { newPlayer = orig }()

	newPlayer = func(instance string, source playerctl.Source) (*playerctl.Player, error) {
		return nil, errors.New("connect failed")
	}

	var out, errOut bytes.Buffer
	code := run([]string{"--player", "vlc", "status"}, &out, &errOut)
	if code != 1 || !strings.Contains(errOut.String(), "failed to connect") {
		t.Fatalf("expected connection failure path: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestSelectInstances(t *testing.T) {
	got := selectInstances("vlc, spotify", "spotify", false)
	if len(got) != 1 || got[0] != "vlc" {
		t.Fatalf("selectInstances mismatch: %#v", got)
	}
}

func TestRunListAll(t *testing.T) {
	origManager := newPlayerManger
	defer func() { newPlayerManger = origManager }()
	newPlayerManger = func(source playerctl.Source) (*playerctl.PlayerManager, error) {
		m := &playerctl.PlayerManager{}
		return m, nil
	}
	var out, errOut bytes.Buffer
	code := run([]string{"--list-all"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected success list-all code=%d err=%q", code, errOut.String())
	}
}

func TestRunFollowValidation(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--player", "vlc", "--follow", "play"}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "only supported") {
		t.Fatalf("follow validation failed code=%d err=%q", code, errOut.String())
	}
}
