package playerctl

import (
	"testing"
)

func TestPlaybackStatusString(t *testing.T) {
	tests := []struct {
		status   PlaybackStatus
		expected string
	}{
		{PlaybackStatusPlaying, "Playing"},
		{PlaybackStatusPaused, "Paused"},
		{PlaybackStatusStopped, "Stopped"},
		{PlaybackStatus(99), "Unknown"},
	}

	for _, test := range tests {
		if got := test.status.String(); got != test.expected {
			t.Errorf("PlaybackStatus(%d).String() = %q, want %q", test.status, got, test.expected)
		}
	}
}

func TestLoopStatusString(t *testing.T) {
	tests := []struct {
		status   LoopStatus
		expected string
	}{
		{LoopStatusNone, "None"},
		{LoopStatusTrack, "Track"},
		{LoopStatusPlaylist, "Playlist"},
		{LoopStatus(99), "Unknown"},
	}

	for _, test := range tests {
		if got := test.status.String(); got != test.expected {
			t.Errorf("LoopStatus(%d).String() = %q, want %q", test.status, got, test.expected)
		}
	}
}

func TestSourceString(t *testing.T) {
	tests := []struct {
		source   Source
		expected string
	}{
		{SourceNone, "None"},
		{SourceDBusSession, "DBusSession"},
		{SourceDBusSystem, "DBusSystem"},
		{Source(99), "Unknown"},
	}

	for _, test := range tests {
		if got := test.source.String(); got != test.expected {
			t.Errorf("Source(%d).String() = %q, want %q", test.source, got, test.expected)
		}
	}
}
