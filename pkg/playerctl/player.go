package playerctl

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	mprisPath       = "/org/mpris/MediaPlayer2"
	propertiesIface = "org.freedesktop.DBus.Properties"
	playerIface     = "org.mpris.MediaPlayer2.Player"
	dbusIface       = "org.freedesktop.DBus"
)

var (
	connectSessionBus = dbus.ConnectSessionBus
	connectSystemBus  = dbus.ConnectSystemBus
)

// Player models a single controllable media player connection.
type Player struct {
	mu sync.RWMutex

	name     string
	instance string
	source   Source

	conn       *dbus.Conn
	obj        dbus.BusObject
	signalChan chan *dbus.Signal
	done       chan struct{}

	disappeared bool
	closed      bool
}

// NewPlayer creates a Player from an instance string and source.
func NewPlayer(instance string, source Source) (*Player, error) {
	if instance == "" {
		return nil, ErrPlayerNotFound
	}

	pn := NewPlayerName(instance, source)
	return NewPlayerFromName(pn)
}

// NewPlayerFromName creates a Player from a fully qualified PlayerName.
func NewPlayerFromName(name *PlayerName) (*Player, error) {
	if name == nil || name.Instance == "" {
		return nil, ErrPlayerNotFound
	}

	p := &Player{
		name:       name.Name,
		instance:   name.Instance,
		source:     name.Source,
		signalChan: make(chan *dbus.Signal, 32),
		done:       make(chan struct{}),
	}

	if err := p.connect(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Player) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.source == SourceNone {
		return nil
	}

	var (
		conn *dbus.Conn
		err  error
	)
	if p.source == SourceDBusSession {
		conn, err = connectSessionBus()
	} else {
		conn, err = connectSystemBus()
	}
	if err != nil {
		return fmt.Errorf("connect dbus: %w", err)
	}

	p.conn = conn
	p.obj = conn.Object("org.mpris.MediaPlayer2."+p.instance, dbus.ObjectPath(mprisPath))

	if err := p.subscribeSignalsLocked(); err != nil {
		_ = conn.Close()
		p.conn = nil
		return err
	}

	go p.signalLoop()
	return nil
}

func (p *Player) subscribeSignalsLocked() error {
	if p.conn == nil {
		return nil
	}
	matches := []string{
		"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='" + mprisPath + "'",
		"type='signal',interface='org.mpris.MediaPlayer2.Player',member='Seeked',path='" + mprisPath + "'",
		"type='signal',interface='org.freedesktop.DBus',member='NameOwnerChanged',arg0='org.mpris.MediaPlayer2." + p.instance + "'",
	}
	for _, rule := range matches {
		if call := p.conn.BusObject().Call(dbusIface+".AddMatch", 0, rule); call.Err != nil {
			return fmt.Errorf("subscribe signal %q: %w", rule, call.Err)
		}
	}
	p.conn.Signal(p.signalChan)
	return nil
}

func (p *Player) signalLoop() {
	for {
		select {
		case <-p.done:
			return
		case sig := <-p.signalChan:
			if sig == nil {
				continue
			}
			if sig.Name == dbusIface+".NameOwnerChanged" && len(sig.Body) >= 3 {
				if name, ok := sig.Body[0].(string); ok && name == "org.mpris.MediaPlayer2."+p.instance {
					if newOwner, ok := sig.Body[2].(string); ok && newOwner == "" {
						p.mu.Lock()
						p.disappeared = true
						p.mu.Unlock()
					}
				}
			}
		}
	}
}

// Close marks the Player as closed and releases locally held state.
func (p *Player) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	close(p.done)
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
}

// Closed reports whether the player has been closed.
func (p *Player) Closed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

// Disappeared reports whether the player service vanished from D-Bus.
func (p *Player) Disappeared() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.disappeared
}

// Name returns the short player name.
func (p *Player) Name() string { p.mu.RLock(); defer p.mu.RUnlock(); return p.name }

// Instance returns the full player instance.
func (p *Player) Instance() string { p.mu.RLock(); defer p.mu.RUnlock(); return p.instance }

// Source returns the bus source used by this player.
func (p *Player) Source() Source { p.mu.RLock(); defer p.mu.RUnlock(); return p.source }

func (p *Player) getProperty(name string, out any) error {
	p.mu.RLock()
	obj := p.obj
	p.mu.RUnlock()
	if obj == nil {
		return ErrPlayerNotFound
	}
	v, err := obj.GetProperty(playerIface + "." + name)
	if err != nil {
		return err
	}
	return dbus.Store([]interface{}{v.Value()}, out)
}

// PlaybackStatus returns the player's current playback status.
func (p *Player) PlaybackStatus() (PlaybackStatus, error) {
	var raw string
	if err := p.getProperty("PlaybackStatus", &raw); err != nil {
		return PlaybackStatusStopped, err
	}
	status, ok := ParsePlaybackStatus(raw)
	if !ok {
		return PlaybackStatusStopped, FormatError{Message: "invalid playback status"}
	}
	return status, nil
}

// LoopStatus returns the player's loop status.
func (p *Player) LoopStatus() (LoopStatus, error) {
	var raw string
	if err := p.getProperty("LoopStatus", &raw); err != nil {
		return LoopStatusNone, err
	}
	status, ok := ParseLoopStatus(raw)
	if !ok {
		return LoopStatusNone, FormatError{Message: "invalid loop status"}
	}
	return status, nil
}

// Shuffle returns whether shuffle is enabled.
func (p *Player) Shuffle() (bool, error) { var v bool; return v, p.getProperty("Shuffle", &v) }

// Volume returns player volume.
func (p *Player) Volume() (float64, error) { var v float64; return v, p.getProperty("Volume", &v) }

// Position returns current track position in microseconds.
func (p *Player) Position() (int64, error) { var v int64; return v, p.getProperty("Position", &v) }

// Metadata returns metadata map values.
func (p *Player) Metadata() (map[string]dbus.Variant, error) {
	var meta map[string]dbus.Variant
	if err := p.getProperty("Metadata", &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// CanControl reports whether the player supports control actions.
func (p *Player) CanControl() (bool, error) { var v bool; return v, p.getProperty("CanControl", &v) }
func (p *Player) CanPlay() (bool, error)    { var v bool; return v, p.getProperty("CanPlay", &v) }
func (p *Player) CanPause() (bool, error)   { var v bool; return v, p.getProperty("CanPause", &v) }
func (p *Player) CanSeek() (bool, error)    { var v bool; return v, p.getProperty("CanSeek", &v) }
func (p *Player) CanGoNext() (bool, error)  { var v bool; return v, p.getProperty("CanGoNext", &v) }
func (p *Player) CanGoPrevious() (bool, error) {
	var v bool
	return v, p.getProperty("CanGoPrevious", &v)
}

func (p *Player) callPlayer(method string, args ...any) error {
	p.mu.RLock()
	obj := p.obj
	p.mu.RUnlock()
	if obj == nil {
		return ErrPlayerNotFound
	}
	if call := obj.Call(playerIface+"."+method, 0, args...); call.Err != nil {
		return InvalidCommandError{Command: strings.ToLower(method)}
	}
	return nil
}

// Play starts playback.
func (p *Player) Play() error      { return p.callPlayer("Play") }
func (p *Player) Pause() error     { return p.callPlayer("Pause") }
func (p *Player) PlayPause() error { return p.callPlayer("PlayPause") }
func (p *Player) Stop() error      { return p.callPlayer("Stop") }
func (p *Player) Next() error      { return p.callPlayer("Next") }
func (p *Player) Previous() error  { return p.callPlayer("Previous") }
func (p *Player) Seek(offset int64) error {
	return p.callPlayer("Seek", offset)
}
func (p *Player) SetPosition(trackID string, position int64) error {
	return p.callPlayer("SetPosition", dbus.ObjectPath(trackID), position)
}
func (p *Player) OpenUri(uri string) error { return p.callPlayer("OpenUri", uri) }

// OpenURI is a compatibility alias for OpenUri.
func (p *Player) OpenURI(uri string) error { return p.OpenUri(uri) }

// SetVolume sets the player volume.
func (p *Player) SetVolume(volume float64) error {
	return p.setProperty("Volume", dbus.MakeVariant(volume))
}

// SetLoopStatus sets loop mode.
func (p *Player) SetLoopStatus(status LoopStatus) error {
	return p.setProperty("LoopStatus", dbus.MakeVariant(status.String()))
}

// SetShuffle toggles shuffle mode.
func (p *Player) SetShuffle(enabled bool) error {
	return p.setProperty("Shuffle", dbus.MakeVariant(enabled))
}

func (p *Player) setProperty(name string, value dbus.Variant) error {
	p.mu.RLock()
	obj := p.obj
	p.mu.RUnlock()
	if obj == nil {
		return ErrPlayerNotFound
	}
	if call := obj.Call(propertiesIface+".Set", 0, playerIface, name, value); call.Err != nil {
		return call.Err
	}
	return nil
}

// ExtractArtist returns a best-effort artist string from a metadata map.
func ExtractArtist(meta map[string]dbus.Variant) string {
	v, ok := meta["xesam:artist"]
	if !ok {
		return ""
	}
	switch artists := v.Value().(type) {
	case []string:
		return strings.Join(artists, ", ")
	case string:
		return artists
	case []interface{}:
		parts := make([]string, 0, len(artists))
		for _, a := range artists {
			if s, ok := a.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return ""
	}
}

func extractStringKey(meta map[string]dbus.Variant, key string) string {
	if v, ok := meta[key]; ok {
		if s, ok := v.Value().(string); ok {
			return s
		}
	}
	return ""
}

// ExtractTitle returns the xesam:title string from a metadata map.
func ExtractTitle(meta map[string]dbus.Variant) string {
	return extractStringKey(meta, "xesam:title")
}

// ExtractAlbum returns the xesam:album string from a metadata map.
func ExtractAlbum(meta map[string]dbus.Variant) string {
	return extractStringKey(meta, "xesam:album")
}

// ExtractTrackID returns the mpris:trackid string from a metadata map.
func ExtractTrackID(meta map[string]dbus.Variant) string {
	if v, ok := meta["mpris:trackid"]; ok {
		switch t := v.Value().(type) {
		case dbus.ObjectPath:
			return string(t)
		case string:
			return t
		}
	}
	return ""
}

// GetArtist returns a best-effort artist string from metadata.
func (p *Player) GetArtist() (string, error) {
	meta, err := p.Metadata()
	if err != nil {
		return "", err
	}
	return ExtractArtist(meta), nil
}

func (p *Player) metadataStringKey(key string) (string, error) {
	meta, err := p.Metadata()
	if err != nil {
		return "", err
	}
	return extractStringKey(meta, key), nil
}

// GetTitle returns xesam:title metadata.
func (p *Player) GetTitle() (string, error) { return p.metadataStringKey("xesam:title") }

// GetAlbum returns xesam:album metadata.
func (p *Player) GetAlbum() (string, error) { return p.metadataStringKey("xesam:album") }

// GetTrackID returns mpris:trackid metadata.
func (p *Player) GetTrackID() (string, error) {
	meta, err := p.Metadata()
	if err != nil {
		return "", err
	}
	return ExtractTrackID(meta), nil
}

// WaitForDisappear waits for the player to vanish or timeout.
func (p *Player) WaitForDisappear(timeout time.Duration) bool {
	end := time.NewTimer(timeout)
	defer end.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-end.C:
			return false
		case <-ticker.C:
			if p.Disappeared() {
				return true
			}
		}
	}
}
