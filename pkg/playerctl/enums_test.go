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

func TestParsePlaybackStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected PlaybackStatus
		ok       bool
	}{
		{"Playing", PlaybackStatusPlaying, true},
		{"playing", PlaybackStatusPlaying, true},
		{"Paused", PlaybackStatusPaused, true},
		{"paused", PlaybackStatusPaused, true},
		{"Stopped", PlaybackStatusStopped, true},
		{"stopped", PlaybackStatusStopped, true},
		{"Unknown", PlaybackStatusStopped, false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, ok := ParsePlaybackStatus(test.input)
			if got != test.expected || ok != test.ok {
				t.Errorf("ParsePlaybackStatus(%q) = %v, %v, want %v, %v", test.input, got, ok, test.expected, test.ok)
			}
		})
	}
}

func TestParseLoopStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected LoopStatus
		ok       bool
	}{
		{"None", LoopStatusNone, true},
		{"none", LoopStatusNone, true},
		{"Track", LoopStatusTrack, true},
		{"track", LoopStatusTrack, true},
		{"Playlist", LoopStatusPlaylist, true},
		{"playlist", LoopStatusPlaylist, true},
		{"Unknown", LoopStatusNone, false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, ok := ParseLoopStatus(test.input)
			if got != test.expected || ok != test.ok {
				t.Errorf("ParseLoopStatus(%q) = %v, %v, want %v, %v", test.input, got, ok, test.expected, test.ok)
			}
		})
	}
}
