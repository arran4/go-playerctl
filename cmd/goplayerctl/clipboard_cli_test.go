package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/arran4/go-playerctl/pkg/playerctl"
)

type fakeCopier struct {
	calls []string
	err   error
}

func (f *fakeCopier) Copy(text string) error {
	f.calls = append(f.calls, text)
	return f.err
}

func withCLIFakes(t *testing.T) *fakeCopier {
	t.Helper()
	originalCopier := defaultClipboardCopier
	originalSelect := selectPlayers
	originalRender := renderCommand
	originalDump := renderDump
	originalURL := lookupURL
	copier := &fakeCopier{}
	defaultClipboardCopier = copier
	selectPlayers = func(_, _ []string, _, _ bool) []string { return []string{"one"} }
	renderCommand = func(_ string, _ string, stdout, _ io.Writer, _ cliOptions, _ []string) int {
		_, _ = io.WriteString(stdout, "Björk — 世界\n")
		return 0
	}
	t.Cleanup(func() {
		defaultClipboardCopier = originalCopier
		selectPlayers = originalSelect
		renderCommand = originalRender
		renderDump = originalDump
		lookupURL = originalURL
	})
	return copier
}

func TestCopyUsesExactFinalOutputOnceAndPreservesStdout(t *testing.T) {
	copier := withCLIFakes(t)
	var stdout, stderr bytes.Buffer
	code := run([]string{"--copy", "title"}, &stdout, &stderr)
	if code != 0 || stdout.String() != "Björk — 世界\n" || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !reflect.DeepEqual(copier.calls, []string{"Björk — 世界\n"}) {
		t.Fatalf("copy calls = %#v", copier.calls)
	}
}

func TestCopyFailurePreservesStdoutAndFails(t *testing.T) {
	copier := withCLIFakes(t)
	copier.err = errors.New("clipboard unavailable")
	var stdout, stderr bytes.Buffer
	code := run([]string{"--copy", "status"}, &stdout, &stderr)
	if code == 0 || stdout.String() != "Björk — 世界\n" || !strings.Contains(stderr.String(), "clipboard unavailable") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestNilCopierIsAnError(t *testing.T) {
	withCLIFakes(t)
	defaultClipboardCopier = nil
	if err := copyToClipboard("value"); err == nil {
		t.Fatal("nil copier unexpectedly succeeded")
	}
}

func TestFailedCommandDoesNotCopy(t *testing.T) {
	copier := withCLIFakes(t)
	renderCommand = func(_ string, _ string, _, stderr io.Writer, _ cliOptions, _ []string) int {
		fmt.Fprintln(stderr, "query failed")
		return 1
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--copy", "status"}, &stdout, &stderr); code == 0 {
		t.Fatal("failed command returned success")
	}
	if len(copier.calls) != 0 {
		t.Fatalf("copier called on failure: %#v", copier.calls)
	}
}

func TestAllPlayersCopyAggregatesOnce(t *testing.T) {
	copier := withCLIFakes(t)
	selectPlayers = func(_, _ []string, _, _ bool) []string { return []string{"foo", "bar"} }
	renderCommand = func(_ string, instance string, stdout, _ io.Writer, opts cliOptions, _ []string) int {
		fmt.Fprintf(stdout, "%s %s-value\n", instance, instance)
		if !opts.allPlayers {
			t.Error("allPlayers option was not retained")
		}
		return 0
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--all-players", "--copy", "status"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	want := "foo foo-value\nbar bar-value\n"
	if stdout.String() != want || !reflect.DeepEqual(copier.calls, []string{want}) {
		t.Fatalf("stdout=%q calls=%#v", stdout.String(), copier.calls)
	}
}

func TestDumpCommandsUseCopyPipeline(t *testing.T) {
	for _, command := range []string{"dump", "dump-json"} {
		t.Run(command, func(t *testing.T) {
			copier := withCLIFakes(t)
			renderDump = func(_ []string, stdout, _ io.Writer, opts cliOptions) int {
				fmt.Fprintf(stdout, "dump json=%v\n", opts.json)
				return 0
			}
			var stdout, stderr bytes.Buffer
			if code := run([]string{"--copy", command}, &stdout, &stderr); code != 0 {
				t.Fatalf("code=%d stderr=%q", code, stderr.String())
			}
			if len(copier.calls) != 1 || copier.calls[0] != stdout.String() {
				t.Fatalf("stdout=%q calls=%#v", stdout.String(), copier.calls)
			}
		})
	}
}

func TestCopyRejectedForNonFiniteCommands(t *testing.T) {
	for _, args := range [][]string{
		{"--copy", "tui"},
		{"--copy", "daemon"},
		{"--copy", "mock"},
		{"--copy", "--follow", "status"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(args, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "--copy") {
				t.Fatalf("code=%d stderr=%q", code, stderr.String())
			}
		})
	}
}

func TestSelectInstancesRankingAndIgnore(t *testing.T) {
	originalDiscover, originalStatus := discoverPlayers, lookupStatus
	discoverPlayers = func() ([]string, error) { return []string{"stopped", "paused", "playing", "ignored"}, nil }
	statuses := map[string]playerctl.PlaybackStatus{
		"stopped": playerctl.PlaybackStatusStopped,
		"paused":  playerctl.PlaybackStatusPaused,
		"playing": playerctl.PlaybackStatusPlaying,
		"ignored": playerctl.PlaybackStatusPlaying,
	}
	lookupStatus = func(instance string) (playerctl.PlaybackStatus, error) { return statuses[instance], nil }
	t.Cleanup(func() { discoverPlayers, lookupStatus = originalDiscover, originalStatus })
	got := selectInstances(nil, []string{"ignored"}, false, true)
	want := []string{"playing", "paused", "stopped"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranked instances=%#v want=%#v", got, want)
	}
}

func TestURLSelectionSemanticsAndSchemes(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		instances []string
		urls      map[string]string
		want      string
		calls     []string
		wantCode  int
	}{
		{"automatic preferred", []string{"url"}, []string{"playing", "paused"}, map[string]string{"playing": "https://example/α"}, "https://example/α\n", []string{"playing"}, 0},
		{"automatic fallback", []string{"url"}, []string{"playing", "paused"}, map[string]string{"paused": "spotify:track:123"}, "spotify:track:123\n", []string{"playing", "paused"}, 0},
		{"explicit single", []string{"--player", "foo", "url"}, []string{"foo"}, map[string]string{"foo": "file:///tmp/song"}, "file:///tmp/song\n", []string{"foo"}, 0},
		{"explicit ordered", []string{"--player", "foo,bar", "url"}, []string{"foo", "bar"}, map[string]string{"bar": "http://example"}, "http://example\n", []string{"foo", "bar"}, 0},
		{"automatic failure", []string{"url"}, []string{"foo", "bar"}, nil, "", []string{"foo", "bar"}, 1},
		{"explicit failure", []string{"--player", "foo", "url"}, []string{"foo"}, nil, "", []string{"foo"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withCLIFakes(t)
			selectPlayers = func(_, _ []string, _, _ bool) []string { return tt.instances }
			var calls []string
			lookupURL = func(instance string) (string, error) {
				calls = append(calls, instance)
				return tt.urls[instance], nil
			}
			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != tt.wantCode || stdout.String() != tt.want || !reflect.DeepEqual(calls, tt.calls) {
				t.Fatalf("code=%d stdout=%q stderr=%q calls=%#v", code, stdout.String(), stderr.String(), calls)
			}
		})
	}
}

func TestAllPlayersURLRetainsAggregateAndCopiesOnce(t *testing.T) {
	copier := withCLIFakes(t)
	selectPlayers = func(_, _ []string, _, _ bool) []string { return []string{"foo", "bar"} }
	lookupURL = func(instance string) (string, error) {
		return map[string]string{"foo": "spotify:x", "bar": "file:///音楽"}[instance], nil
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--all-players", "--copy", "url"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	want := "foo spotify:x\nbar file:///音楽\n"
	if stdout.String() != want || !reflect.DeepEqual(copier.calls, []string{want}) {
		t.Fatalf("stdout=%q calls=%#v", stdout.String(), copier.calls)
	}
}
