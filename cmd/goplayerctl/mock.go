package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

type MockPlayer struct {
	props *prop.Properties
}

func (m *MockPlayer) Next() *dbus.Error {
	fmt.Println("Mock: Next")
	return nil
}

func (m *MockPlayer) Previous() *dbus.Error {
	fmt.Println("Mock: Previous")
	return nil
}

func (m *MockPlayer) Pause() *dbus.Error {
	fmt.Println("Mock: Pause")
	m.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Paused")
	return nil
}

func (m *MockPlayer) PlayPause() *dbus.Error {
	fmt.Println("Mock: PlayPause")
	v, err := m.props.Get("org.mpris.MediaPlayer2.Player", "PlaybackStatus")
	if err != nil {
		return nil
	}
	status := v.Value().(string)
	if status == "Playing" {
		m.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Paused")
	} else {
		m.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Playing")
	}
	return nil
}

func (m *MockPlayer) Stop() *dbus.Error {
	fmt.Println("Mock: Stop")
	m.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Stopped")
	return nil
}

func (m *MockPlayer) Play() *dbus.Error {
	fmt.Println("Mock: Play")
	m.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", "Playing")
	return nil
}

func (m *MockPlayer) Seek(offset int64) *dbus.Error {
	fmt.Printf("Mock: Seek %d\n", offset)
	return nil
}

func (m *MockPlayer) SetPosition(trackId dbus.ObjectPath, position int64) *dbus.Error {
	fmt.Printf("Mock: SetPosition %s %d\n", trackId, position)
	return nil
}

func (m *MockPlayer) OpenUri(uri string) *dbus.Error {
	fmt.Printf("Mock: OpenUri %s\n", uri)
	return nil
}

func (m *MockPlayer) Raise() *dbus.Error {
	fmt.Println("Mock: Raise")
	return nil
}

func (m *MockPlayer) Quit() *dbus.Error {
	fmt.Println("Mock: Quit")
	os.Exit(0)
	return nil
}

func runMock(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mock", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintf(stderr, "failed to connect to dbus: %v\n", err)
		return 1
	}
	defer conn.Close()

	reply, err := conn.RequestName("org.mpris.MediaPlayer2.mock", dbus.NameFlagDoNotQueue)
	if err != nil {
		fmt.Fprintf(stderr, "failed to request name: %v\n", err)
		return 1
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintf(stderr, "name already owned\n")
		return 1
	}

	mock := &MockPlayer{}

	err = conn.Export(mock, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	if err != nil {
		fmt.Fprintf(stderr, "failed to export player interface: %v\n", err)
		return 1
	}

	err = conn.Export(mock, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2")
	if err != nil {
		fmt.Fprintf(stderr, "failed to export root interface: %v\n", err)
		return 1
	}

	propsSpec := map[string]map[string]*prop.Prop{
		"org.mpris.MediaPlayer2": {
			"CanQuit": {
				Value:    true,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"HasTrackList": {
				Value:    false,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"Identity": {
				Value:    "MockPlayer",
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
		},
		"org.mpris.MediaPlayer2.Player": {
			"PlaybackStatus": {
				Value:    "Playing",
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"LoopStatus": {
				Value:    "None",
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"Rate": {
				Value:    1.0,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"Shuffle": {
				Value:    false,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"Metadata": {
				Value: map[string]dbus.Variant{
					"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")),
					"xesam:title":   dbus.MakeVariant("Mock Title"),
					"xesam:artist":  dbus.MakeVariant([]string{"Mock Artist"}),
					"xesam:album":   dbus.MakeVariant("Mock Album"),
				},
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"Volume": {
				Value:    1.0,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"Position": {
				Value:    int64(0),
				Writable: false,
				Emit:     prop.EmitFalse,
				Callback: nil,
			},
			"CanGoNext": {
				Value:    true,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"CanGoPrevious": {
				Value:    true,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"CanPlay": {
				Value:    true,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"CanPause": {
				Value:    true,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"CanSeek": {
				Value:    true,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"CanControl": {
				Value:    true,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
		},
	}
	props, err := prop.Export(conn, "/org/mpris/MediaPlayer2", propsSpec)
	if err != nil {
		fmt.Fprintf(stderr, "failed to export properties: %v\n", err)
		return 1
	}
	mock.props = props

	n := &introspect.Node{
		Name: "/org/mpris/MediaPlayer2",
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name: "org.mpris.MediaPlayer2",
				Methods: []introspect.Method{
					{Name: "Raise"},
					{Name: "Quit"},
				},
				Properties: props.Introspection("org.mpris.MediaPlayer2"),
			},
			{
				Name: "org.mpris.MediaPlayer2.Player",
				Methods: []introspect.Method{
					{Name: "Next"},
					{Name: "Previous"},
					{Name: "Pause"},
					{Name: "PlayPause"},
					{Name: "Stop"},
					{Name: "Play"},
					{
						Name: "Seek",
						Args: []introspect.Arg{
							{Name: "Offset", Type: "x", Direction: "in"},
						},
					},
					{
						Name: "SetPosition",
						Args: []introspect.Arg{
							{Name: "TrackId", Type: "o", Direction: "in"},
							{Name: "Position", Type: "x", Direction: "in"},
						},
					},
					{
						Name: "OpenUri",
						Args: []introspect.Arg{
							{Name: "Uri", Type: "s", Direction: "in"},
						},
					},
				},
				Properties: props.Introspection("org.mpris.MediaPlayer2.Player"),
			},
		},
	}
	err = conn.Export(introspect.NewIntrospectable(n), "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Introspectable")
	if err != nil {
		fmt.Fprintf(stderr, "failed to export introspect: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Mock player running as org.mpris.MediaPlayer2.mock")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	return 0
}
