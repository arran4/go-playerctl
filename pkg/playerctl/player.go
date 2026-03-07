package playerctl

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	mprisPath         = "/org/mpris/MediaPlayer2"
	propertiesIface   = "org.freedesktop.DBus.Properties"
	playerIface       = "org.mpris.MediaPlayer2.Player"
	trackListIface    = "org.mpris.MediaPlayer2.TrackList"
	playlistsIface    = "org.mpris.MediaPlayer2.Playlists"
	dbusIface         = "org.freedesktop.DBus"
)

// Playlist defines the structure of an MPRIS playlist (Id, Name, Icon).
type Playlist struct {
	Id   dbus.ObjectPath
	Name string
	Icon string
}

// ActivePlaylist defines the structure of the ActivePlaylist property.
type ActivePlaylist struct {
	Valid    bool
	Playlist Playlist
}

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
	eventChan  chan *dbus.Signal

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
		eventChan:  make(chan *dbus.Signal, 32),
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
	defer func() {
		if p.eventChan != nil {
			close(p.eventChan)
		}
	}()
	for {
		select {
		case <-p.done:
			return
		case sig := <-p.signalChan:
			if sig == nil {
				continue
			}

			select {
			case p.eventChan <- sig:
			default:
				// Drop signal if eventChan is full
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

// Events returns a read-only channel for D-Bus signals received by the player.
func (p *Player) Events() <-chan *dbus.Signal {
	return p.eventChan
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

func (p *Player) getInterfaceProperty(iface, name string, out any) error {
	p.mu.RLock()
	obj := p.obj
	p.mu.RUnlock()
	if obj == nil {
		return ErrPlayerNotFound
	}
	v, err := obj.GetProperty(iface + "." + name)
	if err != nil {
		return err
	}
	return dbus.Store([]interface{}{v.Value()}, out)
}

func (p *Player) callInterface(iface, method string, args ...any) error {
	p.mu.RLock()
	obj := p.obj
	p.mu.RUnlock()
	if obj == nil {
		return ErrPlayerNotFound
	}
	if call := obj.Call(iface+"."+method, 0, args...); call.Err != nil {
		return InvalidCommandError{Command: strings.ToLower(method)}
	}
	return nil
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

// HasTrackList reports whether the player supports the TrackList interface.
func (p *Player) HasTrackList() (bool, error) { var v bool; return v, p.getInterfaceProperty("org.mpris.MediaPlayer2", "HasTrackList", &v) }

// Tracks returns the current track list as an array of ObjectPaths.
func (p *Player) Tracks() ([]dbus.ObjectPath, error) {
	var v []dbus.ObjectPath
	return v, p.getInterfaceProperty(trackListIface, "Tracks", &v)
}

// CanEditTracks reports whether the track list can be modified.
func (p *Player) CanEditTracks() (bool, error) {
	var v bool
	return v, p.getInterfaceProperty(trackListIface, "CanEditTracks", &v)
}

// GetTracksMetadata retrieves metadata for a given list of track IDs.
func (p *Player) GetTracksMetadata(trackIds []dbus.ObjectPath) ([]map[string]dbus.Variant, error) {
	p.mu.RLock()
	obj := p.obj
	p.mu.RUnlock()
	if obj == nil {
		return nil, ErrPlayerNotFound
	}
	var out []map[string]dbus.Variant
	call := obj.Call(trackListIface+".GetTracksMetadata", 0, trackIds)
	if call.Err != nil {
		return nil, call.Err
	}
	if err := call.Store(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddTrack adds a URI to the tracklist.
func (p *Player) AddTrack(uri string, afterTrack dbus.ObjectPath, setAsCurrent bool) error {
	return p.callInterface(trackListIface, "AddTrack", uri, afterTrack, setAsCurrent)
}

// RemoveTrack removes a track by ID from the tracklist.
func (p *Player) RemoveTrack(trackId dbus.ObjectPath) error {
	return p.callInterface(trackListIface, "RemoveTrack", trackId)
}

// GoTo makes the player switch to the specified track in the tracklist.
func (p *Player) GoTo(trackId dbus.ObjectPath) error {
	return p.callInterface(trackListIface, "GoTo", trackId)
}

// PlaylistCount returns the number of available playlists.
func (p *Player) PlaylistCount() (uint32, error) {
	var v uint32
	return v, p.getInterfaceProperty(playlistsIface, "PlaylistCount", &v)
}

// Orderings returns the available orderings for GetPlaylists.
func (p *Player) Orderings() ([]string, error) {
	var v []string
	return v, p.getInterfaceProperty(playlistsIface, "Orderings", &v)
}

// ActivePlaylist returns the currently active playlist.
func (p *Player) ActivePlaylist() (ActivePlaylist, error) {
	var raw struct {
		Valid    bool
		Playlist struct {
			Id   dbus.ObjectPath
			Name string
			Icon string
		}
	}
	err := p.getInterfaceProperty(playlistsIface, "ActivePlaylist", &raw)
	return ActivePlaylist{
		Valid: raw.Valid,
		Playlist: Playlist{
			Id:   raw.Playlist.Id,
			Name: raw.Playlist.Name,
			Icon: raw.Playlist.Icon,
		},
	}, err
}

// ActivatePlaylist activates the specified playlist.
func (p *Player) ActivatePlaylist(playlistId dbus.ObjectPath) error {
	return p.callInterface(playlistsIface, "ActivatePlaylist", playlistId)
}

// GetPlaylists returns a list of playlists given an index, count, order, and sort direction.
func (p *Player) GetPlaylists(index uint32, maxCount uint32, order string, reverseOrder bool) ([]Playlist, error) {
	p.mu.RLock()
	obj := p.obj
	p.mu.RUnlock()
	if obj == nil {
		return nil, ErrPlayerNotFound
	}

	// The signature is a(oss)
	var raw []struct {
		Id   dbus.ObjectPath
		Name string
		Icon string
	}

	call := obj.Call(playlistsIface+".GetPlaylists", 0, index, maxCount, order, reverseOrder)
	if call.Err != nil {
		return nil, call.Err
	}

	if err := call.Store(&raw); err != nil {
		return nil, err
	}

	var out []Playlist
	for _, item := range raw {
		out = append(out, Playlist{
			Id:   item.Id,
			Name: item.Name,
			Icon: item.Icon,
		})
	}
	return out, nil
}

// Volume returns player volume.
func (p *Player) Volume() (float64, error) { var v float64; return v, p.getProperty("Volume", &v) }

// Position returns current track position in microseconds.
func (p *Player) Position() (int64, error) { var v int64; return v, p.getProperty("Position", &v) }

// Rate returns the player's playback rate.
func (p *Player) Rate() (float64, error) { var v float64; return v, p.getProperty("Rate", &v) }

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

// SetRate sets the playback rate.
func (p *Player) SetRate(rate float64) error {
	return p.setProperty("Rate", dbus.MakeVariant(rate))
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
