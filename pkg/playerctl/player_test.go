package playerctl

import (
	"context"
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestNewPlayer(t *testing.T) {
	p, err := NewPlayer("vlc.instance123", SourceNone)
	if err != nil {
		t.Fatalf("NewPlayer returned error: %v", err)
	}

	if got := p.Name(); got != "vlc" {
		t.Fatalf("Name() = %q, want %q", got, "vlc")
	}
	if got := p.Instance(); got != "vlc.instance123" {
		t.Fatalf("Instance() = %q, want %q", got, "vlc.instance123")
	}
	if got := p.Source(); got != SourceNone {
		t.Fatalf("Source() = %v, want %v", got, SourceNone)
	}
	if p.Closed() {
		t.Fatal("new player should not be closed")
	}
}

func TestNewPlayerInvalid(t *testing.T) {
	if _, err := NewPlayer("", SourceNone); err == nil {
		t.Fatal("expected error for empty instance")
	}

	if _, err := NewPlayerFromName(nil); err == nil {
		t.Fatal("expected error for nil PlayerName")
	}

	if _, err := NewPlayerFromName(&PlayerName{}); err == nil {
		t.Fatal("expected error for empty PlayerName instance")
	}
}

func TestPlayerClose(t *testing.T) {
	p, err := NewPlayerFromName(&PlayerName{Name: "spotify", Instance: "spotify", Source: SourceNone})
	if err != nil {
		t.Fatalf("NewPlayerFromName returned error: %v", err)
	}

	p.Close()
	if !p.Closed() {
		t.Fatal("Closed() = false, want true")
	}
}

type fakeObject struct {
	props map[string]dbus.Variant
	calls []string
	err   error
}

func (f *fakeObject) Call(method string, flags dbus.Flags, args ...any) *dbus.Call {
	f.calls = append(f.calls, method)
	return &dbus.Call{Err: f.err}
}
func (f *fakeObject) CallWithContext(ctx context.Context, method string, flags dbus.Flags, args ...any) *dbus.Call {
	return f.Call(method, flags, args...)
}
func (f *fakeObject) Go(method string, flags dbus.Flags, ch chan *dbus.Call, args ...any) *dbus.Call {
	return f.Call(method, flags, args...)
}
func (f *fakeObject) GoWithContext(ctx context.Context, method string, flags dbus.Flags, ch chan *dbus.Call, args ...any) *dbus.Call {
	return f.Call(method, flags, args...)
}
func (f *fakeObject) AddMatchSignal(iface, member string, options ...dbus.MatchOption) *dbus.Call {
	return &dbus.Call{Err: f.err}
}
func (f *fakeObject) RemoveMatchSignal(iface, member string, options ...dbus.MatchOption) *dbus.Call {
	return &dbus.Call{Err: f.err}
}
func (f *fakeObject) GetProperty(p string) (dbus.Variant, error) {
	if f.err != nil {
		return dbus.Variant{}, f.err
	}
	v, ok := f.props[p]
	if !ok {
		return dbus.MakeVariant(nil), nil
	}
	return v, nil
}
func (f *fakeObject) StoreProperty(p string, value any) error { return f.err }
func (f *fakeObject) SetProperty(p string, v any) error       { return f.err }
func (f *fakeObject) Destination() string                     { return "" }
func (f *fakeObject) Path() dbus.ObjectPath                   { return "/" }

func TestPlayerPropertyAndCommandWithFakeBusObject(t *testing.T) {
	f := &fakeObject{props: map[string]dbus.Variant{
		"org.mpris.MediaPlayer2.Player.PlaybackStatus": dbus.MakeVariant("Playing"),
		"org.mpris.MediaPlayer2.Player.LoopStatus":     dbus.MakeVariant("Track"),
		"org.mpris.MediaPlayer2.Player.Shuffle":        dbus.MakeVariant(true),
		"org.mpris.MediaPlayer2.Player.Volume":         dbus.MakeVariant(0.5),
		"org.mpris.MediaPlayer2.Player.Position":       dbus.MakeVariant(int64(42)),
		"org.mpris.MediaPlayer2.Player.Metadata": dbus.MakeVariant(map[string]dbus.Variant{
			"xesam:title":   dbus.MakeVariant("Title"),
			"xesam:album":   dbus.MakeVariant("Album"),
			"xesam:artist":  dbus.MakeVariant([]string{"Artist"}),
			"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/track/1")),
			"mpris:artUrl":  dbus.MakeVariant("file:///tmp/art.jpg"),
		}),
	}}
	p := &Player{obj: f}

	if st, err := p.PlaybackStatus(); err != nil || st != PlaybackStatusPlaying {
		t.Fatalf("status %v %v", st, err)
	}
	if lp, err := p.LoopStatus(); err != nil || lp != LoopStatusTrack {
		t.Fatalf("loop %v %v", lp, err)
	}
	if sh, err := p.Shuffle(); err != nil || !sh {
		t.Fatalf("shuffle %v %v", sh, err)
	}
	if vol, err := p.Volume(); err != nil || vol != 0.5 {
		t.Fatalf("volume %v %v", vol, err)
	}
	if pos, err := p.Position(); err != nil || pos != 42 {
		t.Fatalf("position %v %v", pos, err)
	}
	if title, _ := p.GetTitle(); title != "Title" {
		t.Fatalf("title %q", title)
	}
	if album, _ := p.GetAlbum(); album != "Album" {
		t.Fatalf("album %q", album)
	}
	if artist, _ := p.GetArtist(); artist != "Artist" {
		t.Fatalf("artist %q", artist)
	}
	if track, _ := p.GetTrackID(); track != "/track/1" {
		t.Fatalf("track %q", track)
	}
	if art, _ := p.GetArtUrl(); art != "file:///tmp/art.jpg" {
		t.Fatalf("artUrl %q", art)
	}

	if err := p.Play(); err != nil {
		t.Fatalf("play err: %v", err)
	}
	if len(f.calls) == 0 {
		t.Fatal("expected command call")
	}
}

func TestPlayerCommandErrorMaps(t *testing.T) {
	p := &Player{obj: &fakeObject{err: errors.New("bad")}}
	if err := p.Play(); err == nil {
		t.Fatal("expected command error")
	}
}
