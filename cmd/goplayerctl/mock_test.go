package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arran4/go-playerctl/pkg/playerctl"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

func setupTestDBus(t *testing.T) (*dbus.Conn, *MockPlayer) {
	t.Helper()

	if _, err := exec.LookPath("dbus-daemon"); err != nil {
		t.Skip("dbus-daemon not found, skipping dbus integration test")
	}

	tmpDir := t.TempDir()
	addrFile := filepath.Join(tmpDir, "dbus_address")
	cmd := exec.Command("dbus-daemon", "--session", "--print-address=1", "--nofork")
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start dbus-daemon: %v", err)
	}

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = os.Remove(addrFile)
	})

	addrChan := make(chan string, 1)
	go func() {
		buf := make([]byte, 1024)
		n, err := outPipe.Read(buf)
		if err != nil {
			return
		}
		addrChan <- strings.TrimSpace(string(buf[:n]))
	}()

	var addr string
	select {
	case addr = <-addrChan:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for dbus-daemon address")
	}

	t.Setenv("DBUS_SESSION_BUS_ADDRESS", addr)

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatalf("failed to connect to test session bus: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	reply, err := conn.RequestName("org.mpris.MediaPlayer2.mock", dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("failed to request bus name: %v (reply: %v)", err, reply)
	}

	mock := &MockPlayer{}
	if err := conn.Export(mock, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player"); err != nil {
		t.Fatalf("export player iface: %v", err)
	}
	if err := conn.Export(mock, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2"); err != nil {
		t.Fatalf("export root iface: %v", err)
	}

	propsSpec := map[string]map[string]*prop.Prop{
		"org.mpris.MediaPlayer2": {
			"CanQuit":      {Value: true, Writable: false, Emit: prop.EmitTrue},
			"HasTrackList": {Value: false, Writable: false, Emit: prop.EmitTrue},
			"Identity":     {Value: "MockPlayer", Writable: false, Emit: prop.EmitTrue},
		},
		"org.mpris.MediaPlayer2.Player": {
			"PlaybackStatus": {Value: "Playing", Writable: true, Emit: prop.EmitTrue},
			"LoopStatus":     {Value: "None", Writable: true, Emit: prop.EmitTrue},
			"Rate":           {Value: 1.0, Writable: true, Emit: prop.EmitTrue},
			"Shuffle":        {Value: false, Writable: true, Emit: prop.EmitTrue},
			"Metadata": {
				Value: map[string]dbus.Variant{
					"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")),
					"xesam:title":   dbus.MakeVariant("Mock Title"),
					"xesam:artist":  dbus.MakeVariant([]string{"Mock Artist"}),
					"xesam:album":   dbus.MakeVariant("Mock Album"),
				},
				Writable: true,
				Emit:     prop.EmitTrue,
			},
			"Volume":        {Value: 1.0, Writable: true, Emit: prop.EmitTrue},
			"Position":      {Value: int64(0), Writable: false, Emit: prop.EmitFalse},
			"CanGoNext":     {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanGoPrevious": {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanPlay":       {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanPause":      {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanSeek":       {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanControl":    {Value: true, Writable: false, Emit: prop.EmitTrue},
		},
	}

	props, err := prop.Export(conn, "/org/mpris/MediaPlayer2", propsSpec)
	if err != nil {
		t.Fatalf("export properties: %v", err)
	}
	mock.props = props

	n := &introspect.Node{
		Name: "/org/mpris/MediaPlayer2",
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       "org.mpris.MediaPlayer2",
				Properties: props.Introspection("org.mpris.MediaPlayer2"),
			},
			{
				Name:       "org.mpris.MediaPlayer2.Player",
				Properties: props.Introspection("org.mpris.MediaPlayer2.Player"),
			},
		},
	}
	if err := conn.Export(introspect.NewIntrospectable(n), "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Introspectable"); err != nil {
		t.Fatalf("export introspectable: %v", err)
	}

	return conn, mock
}

func exportFaultyStatusPlayer(t *testing.T, conn *dbus.Conn) {
	t.Helper()

	reply, err := conn.RequestName("org.mpris.MediaPlayer2.faulty", dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("failed to request faulty player name: %v (reply: %v)", err, reply)
	}

	faulty := &MockPlayer{}
	if err := conn.Export(faulty, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player"); err != nil {
		t.Fatalf("export faulty player iface: %v", err)
	}
	if err := conn.Export(faulty, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2"); err != nil {
		t.Fatalf("export faulty root iface: %v", err)
	}

	faultyPropsSpec := map[string]map[string]*prop.Prop{
		"org.mpris.MediaPlayer2": {
			"Identity": {Value: "FaultyPlayer", Writable: false, Emit: prop.EmitTrue},
		},
		"org.mpris.MediaPlayer2.Player": {
			// Deliberately set an invalid status string that causes ParsePlaybackStatus to return false
			"PlaybackStatus": {Value: "CorruptedStatusValue", Writable: false, Emit: prop.EmitTrue},
			"Metadata": {
				Value: map[string]dbus.Variant{
					"xesam:title":  dbus.MakeVariant("Faulty Track Title"),
					"xesam:artist": dbus.MakeVariant([]string{"Faulty Artist"}),
				},
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}

	props, err := prop.Export(conn, "/org/mpris/MediaPlayer2", faultyPropsSpec)
	if err != nil {
		t.Fatalf("export faulty properties: %v", err)
	}
	faulty.props = props
}

func TestMockMetadataFormattedStatus(t *testing.T) {
	_, mock := setupTestDBus(t)

	// 1. One-shot formatted metadata with Playing status
	var stdout, stderr bytes.Buffer
	code := run([]string{"--player", "mock", "--format", "{{ .status }}", "metadata"}, &stdout, &stderr)
	if code != 0 || stdout.String() != "Playing\n" {
		t.Fatalf("expected Playing, got code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}

	// 2. Unformatted metadata retains default title output
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--player", "mock", "metadata"}, &stdout, &stderr)
	if code != 0 || stdout.String() != "Mock Title\n" {
		t.Fatalf("expected 'Mock Title', got code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}

	// 3. Status transition to Paused
	mock.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Paused")
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--player", "mock", "--format", "{{ .artist }} - {{ .title }} | {{ .status }}", "metadata"}, &stdout, &stderr)
	if code != 0 || stdout.String() != "Mock Artist - Mock Title | Paused\n" {
		t.Fatalf("expected Paused output, got code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}

	// 4. Status transition to Stopped
	mock.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Stopped")
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--player", "mock", "--format", "{{ .status }}", "metadata"}, &stdout, &stderr)
	if code != 0 || stdout.String() != "Stopped\n" {
		t.Fatalf("expected Stopped, got code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}

	// 5. Waybar style JSON output validation
	stdout.Reset()
	stderr.Reset()
	waybarFmt := `{"text": "{{ markup_escape .artist }} - {{ markup_escape .title }}", "tooltip": "Spotify: {{ markup_escape .artist }} - {{ markup_escape .title }}", "alt": "{{ .status }}", "class": "{{ .status }}"}`
	code = run([]string{"--player", "mock", "--format", waybarFmt, "metadata"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &parsed); err != nil {
		t.Fatalf("failed to parse Waybar JSON output: %v (raw: %q)", err, stdout.String())
	}
	if parsed["class"] != "Stopped" || parsed["alt"] != "Stopped" || parsed["text"] != "Mock Artist - Mock Title" {
		t.Fatalf("unexpected parsed JSON: %#v", parsed)
	}

	// 6. JSON with js escaping
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--player", "mock", "--format", `{"class": "{{ js .status }}"}`, "metadata"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}
	parsed = nil
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &parsed); err != nil {
		t.Fatalf("failed to parse js-escaped JSON: %v (raw: %q)", err, stdout.String())
	}
	if parsed["class"] != "Stopped" {
		t.Fatalf("expected class Stopped, got %#v", parsed)
	}
}

type testStepExecutor struct {
	mu    sync.Mutex
	steps []func()
	iter  int
	limit int
}

func (s *testStepExecutor) Query(cmd, instance string, opts cliOptions) (string, error) {
	p, err := newPlayer(instance, playerctl.SourceDBusSession)
	if err != nil {
		return "", err
	}
	defer p.Close()
	return queryOutput(cmd, p, opts)
}

func (s *testStepExecutor) Wait() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.iter < len(s.steps) {
		s.steps[s.iter]()
	}
	s.iter++
	return s.iter < s.limit
}

func TestMockFollowMetadataEmitsOnStatusChange(t *testing.T) {
	_, mock := setupTestDBus(t)

	// Verify that --follow metadata emits a new result when status transitions
	// Playing -> Paused -> Paused (unchanged, deduplicated) -> Playing while title/artist remain unchanged.
	mock.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Playing")

	stepExec := &testStepExecutor{
		steps: []func(){
			func() {
				// Tick 0 -> Tick 1: change to Paused
				mock.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Paused")
			},
			func() {
				// Tick 1 -> Tick 2: stay Paused (should be deduplicated, not emitted again)
			},
			func() {
				// Tick 2 -> Tick 3: change back to Playing
				mock.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Playing")
			},
		},
		limit: 4,
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--player", "mock",
		"--follow",
		"--format", "{{ .artist }} - {{ .title }} | {{ .status }}",
		"metadata",
	}, &stdout, &stderr, stepExec)

	if code != 0 {
		t.Fatalf("follow metadata failed with code %d: %s", code, stderr.String())
	}

	expected := "Mock Artist - Mock Title | Playing\nMock Artist - Mock Title | Paused\nMock Artist - Mock Title | Playing\n"
	if stdout.String() != expected {
		t.Fatalf("expected output:\n%q\ngot:\n%q", expected, stdout.String())
	}
}

type controlledFollowExecutor struct {
	queryFunc func(cmd, instance string, opts cliOptions) (string, error)
	waitFunc  func() bool
}

func (c *controlledFollowExecutor) Query(cmd, instance string, opts cliOptions) (string, error) {
	return c.queryFunc(cmd, instance, opts)
}

func (c *controlledFollowExecutor) Wait() bool {
	return c.waitFunc()
}

func TestFollowEdgeCases(t *testing.T) {
	t.Run("MissingPlayerOneShot", func(t *testing.T) {
		orig := newPlayer
		defer func() { newPlayer = orig }()
		newPlayer = func(instance string, source playerctl.Source) (*playerctl.Player, error) {
			return nil, errors.New("player not found")
		}

		var stdout, stderr bytes.Buffer
		code := run([]string{"--player", "missing", "metadata"}, &stdout, &stderr)
		if code != 1 || !strings.Contains(stderr.String(), "failed to connect") {
			t.Fatalf("expected code 1 on missing player, got %d, stderr=%q", code, stderr.String())
		}
	})

	t.Run("MissingPlayerFollowRecoversOrCleanShutdown", func(t *testing.T) {
		iter := 0
		exec := &controlledFollowExecutor{
			queryFunc: func(cmd, instance string, opts cliOptions) (string, error) {
				return "", errors.New("transient error: player disconnected")
			},
			waitFunc: func() bool {
				iter++
				return iter < 3
			},
		}

		var stdout, stderr bytes.Buffer
		code := run([]string{"--player", "missing", "--follow", "metadata"}, &stdout, &stderr, exec)
		if code != 0 {
			t.Fatalf("expected clean shutdown code 0, got %d: %s", code, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Fatalf("expected no output for missing player in follow, got %q", stdout.String())
		}
		if iter != 3 {
			t.Fatalf("expected 3 iterations, got %d", iter)
		}
	})

	t.Run("FailedStatusReadInMetadataDBus", func(t *testing.T) {
		conn, _ := setupTestDBus(t)
		exportFaultyStatusPlayer(t, conn)

		// 1. Unformatted metadata succeeds and returns track title despite status read failure
		var stdout, stderr bytes.Buffer
		code := run([]string{"--player", "faulty", "metadata"}, &stdout, &stderr)
		if code != 0 || stdout.String() != "Faulty Track Title\n" {
			t.Fatalf("expected title 'Faulty Track Title', got code=%d out=%q err=%q", code, stdout.String(), stderr.String())
		}

		// 2. Formatted metadata with default fallback helper safely falls back without claiming a state
		stdout.Reset()
		stderr.Reset()
		code = run([]string{"--player", "faulty", "--format", `Status: [{{ default .status "Unavailable" }}]`, "metadata"}, &stdout, &stderr)
		if code != 0 || stdout.String() != "Status: [Unavailable]\n" {
			t.Fatalf("expected fallback status, got code=%d out=%q err=%q", code, stdout.String(), stderr.String())
		}

		// 3. Direct status command fails with code 1 because status could not be read
		stdout.Reset()
		stderr.Reset()
		code = run([]string{"--player", "faulty", "status"}, &stdout, &stderr)
		if code != 1 || !strings.Contains(stderr.String(), "invalid playback status") {
			t.Fatalf("expected code 1 on status failure, got code=%d out=%q err=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("CleanShutdownAndDeduplication", func(t *testing.T) {
		ticks := 0
		exec := &controlledFollowExecutor{
			queryFunc: func(cmd, instance string, opts cliOptions) (string, error) {
				return "Artist - Title | Playing", nil
			},
			waitFunc: func() bool {
				ticks++
				return ticks < 5
			},
		}

		var stdout, stderr bytes.Buffer
		code := run([]string{"--player", "dummy", "--follow", "metadata"}, &stdout, &stderr, exec)
		if code != 0 {
			t.Fatalf("expected code 0, got %d", code)
		}
		// Over 5 ticks with identical output, it must only emit once
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 1 || lines[0] != "Artist - Title | Playing" {
			t.Fatalf("expected exactly 1 deduplicated line, got %d: %#v", len(lines), lines)
		}
		if ticks != 5 {
			t.Fatalf("expected 5 ticks, got %d", ticks)
		}
	})
}
