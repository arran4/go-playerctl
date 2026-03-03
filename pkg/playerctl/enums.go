package playerctl

import "strings"

// PlaybackStatus represents the playback status for a Player.
type PlaybackStatus int

const (
	// PlaybackStatusPlaying indicates active playback.
	PlaybackStatusPlaying PlaybackStatus = iota
	// PlaybackStatusPaused indicates playback is paused.
	PlaybackStatusPaused
	// PlaybackStatusStopped indicates playback has stopped.
	PlaybackStatusStopped
)

// String returns the string representation of the PlaybackStatus.
func (s PlaybackStatus) String() string {
	switch s {
	case PlaybackStatusPlaying:
		return "Playing"
	case PlaybackStatusPaused:
		return "Paused"
	case PlaybackStatusStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

// LoopStatus represents the loop status for a PlayerctlPlayer.
type LoopStatus int

const (
	// LoopStatusNone means playback stops when no tracks remain.
	LoopStatusNone LoopStatus = iota
	// LoopStatusTrack means the current track repeats.
	LoopStatusTrack
	// LoopStatusPlaylist means the active playlist repeats.
	LoopStatusPlaylist
)

// String returns the string representation of the LoopStatus.
func (s LoopStatus) String() string {
	switch s {
	case LoopStatusNone:
		return "None"
	case LoopStatusTrack:
		return "Track"
	case LoopStatusPlaylist:
		return "Playlist"
	default:
		return "Unknown"
	}
}

// Source represents the source of the name used to control the player.
type Source int

const (
	// SourceNone is used for uninitialized players; source is chosen automatically.
	SourceNone Source = iota
	// SourceDBusSession selects the D-Bus session bus.
	SourceDBusSession
	// SourceDBusSystem selects the D-Bus system bus.
	SourceDBusSystem
)

// String returns the string representation of the Source.
func (s Source) String() string {
	switch s {
	case SourceNone:
		return "None"
	case SourceDBusSession:
		return "DBusSession"
	case SourceDBusSystem:
		return "DBusSystem"
	default:
		return "Unknown"
	}
}

// ParsePlaybackStatus parses a string into a PlaybackStatus.
// Returns the parsed status and a boolean indicating success.
func ParsePlaybackStatus(statusStr string) (PlaybackStatus, bool) {
	switch strings.ToLower(statusStr) {
	case "playing":
		return PlaybackStatusPlaying, true
	case "paused":
		return PlaybackStatusPaused, true
	case "stopped":
		return PlaybackStatusStopped, true
	default:
		return PlaybackStatusStopped, false
	}
}

// ParseLoopStatus parses a string into a LoopStatus.
// Returns the parsed status and a boolean indicating success.
func ParseLoopStatus(statusStr string) (LoopStatus, bool) {
	switch strings.ToLower(statusStr) {
	case "none":
		return LoopStatusNone, true
	case "track":
		return LoopStatusTrack, true
	case "playlist":
		return LoopStatusPlaylist, true
	default:
		return LoopStatusNone, false
	}
}
