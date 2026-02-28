package playerctl

import "strings"



// PlaybackStatus represents the playback status for a PlayerctlPlayer.
type PlaybackStatus int

const (
	PlaybackStatusPlaying PlaybackStatus = iota // Playing
	PlaybackStatusPaused                        // Paused
	PlaybackStatusStopped                       // Stopped
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
	LoopStatusNone     LoopStatus = iota // The playback will stop when there are no more tracks to play.
	LoopStatusTrack                      // The current track will start again from the beginning once it has finished playing.
	LoopStatusPlaylist                   // The playlist will start again from the beginning once it has finished playing.
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
	SourceNone        Source = iota // Only for uninitialized players. Source will be chosen automatically.
	SourceDBusSession               // The player is on the DBus session bus.
	SourceDBusSystem                // The player is on the DBus system bus.
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
