package playerctl

import (
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestCalculateCachedPosition(t *testing.T) {
	p := &Player{}
	basePosition := int64(1000000) // 1 second in microseconds

	// Test paused (no time offset)
	result := p.calculateCachedPosition(PlaybackStatusPaused, time.Now().Add(-time.Second), basePosition)
	if result != basePosition {
		t.Errorf("calculateCachedPosition(Paused) = %d, want %d", result, basePosition)
	}

	// Test stopped (returns 0)
	result = p.calculateCachedPosition(PlaybackStatusStopped, time.Now(), basePosition)
	if result != 0 {
		t.Errorf("calculateCachedPosition(Stopped) = %d, want 0", result)
	}

	// Test playing (adds time offset)
	monotonicTime := time.Now().Add(-2 * time.Second) // 2 seconds ago
	result = p.calculateCachedPosition(PlaybackStatusPlaying, monotonicTime, basePosition)
	expectedPosition := basePosition + 2000000 // base + 2 seconds

	// Allow a small margin of error for execution time (e.g., 50ms = 50,000us)
	if result < expectedPosition || result > expectedPosition+50000 {
		t.Errorf("calculateCachedPosition(Playing) = %d, want ~%d", result, expectedPosition)
	}
}

func TestMetadataGetters(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]dbus.Variant
		artist   string
		title    string
		album    string
		trackID  string
	}{
		{
			name: "All fields present",
			metadata: map[string]dbus.Variant{
				"xesam:artist":  dbus.MakeVariant([]string{"Artist1", "Artist2"}),
				"xesam:title":   dbus.MakeVariant("Test Title"),
				"xesam:album":   dbus.MakeVariant("Test Album"),
				"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")),
			},
			artist:  "Artist1, Artist2",
			title:   "Test Title",
			album:   "Test Album",
			trackID: "/org/mpris/MediaPlayer2/TrackList/NoTrack",
		},
		{
			name: "TrackID as string",
			metadata: map[string]dbus.Variant{
				"mpris:trackid": dbus.MakeVariant("test-track-id"),
			},
			artist:  "",
			title:   "",
			album:   "",
			trackID: "test-track-id",
		},
		{
			name:     "Empty metadata",
			metadata: map[string]dbus.Variant{},
			artist:   "",
			title:    "",
			album:    "",
			trackID:  "",
		},
		{
			name: "Invalid types",
			metadata: map[string]dbus.Variant{
				"xesam:artist":  dbus.MakeVariant("Not an array"),
				"xesam:title":   dbus.MakeVariant(123),
				"xesam:album":   dbus.MakeVariant(456.7),
				"mpris:trackid": dbus.MakeVariant(true),
			},
			artist:  "",
			title:   "",
			album:   "",
			trackID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetArtist(tt.metadata); got != tt.artist {
				t.Errorf("GetArtist() = %v, want %v", got, tt.artist)
			}
			if got := GetTitle(tt.metadata); got != tt.title {
				t.Errorf("GetTitle() = %v, want %v", got, tt.title)
			}
			if got := GetAlbum(tt.metadata); got != tt.album {
				t.Errorf("GetAlbum() = %v, want %v", got, tt.album)
			}
			if got := GetTrackID(tt.metadata); got != tt.trackID {
				t.Errorf("GetTrackID() = %v, want %v", got, tt.trackID)
			}
		})
	}
}
