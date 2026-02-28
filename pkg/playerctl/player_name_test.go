package playerctl

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed testdata/player_names.txt
var playerNamesData string

func TestNewPlayerName(t *testing.T) {
	lines := strings.Split(strings.TrimSpace(playerNamesData), "\n")

	tests := []struct {
		instance string
		source   Source
		expected *PlayerName
	}{
		{lines[0], SourceDBusSession, &PlayerName{Name: "vlc", Instance: "vlc.instance123", Source: SourceDBusSession}},
		{lines[1], SourceDBusSession, &PlayerName{Name: "spotify", Instance: "spotify", Source: SourceDBusSession}},
		{lines[2], SourceDBusSystem, &PlayerName{Name: "mpd", Instance: "mpd", Source: SourceDBusSystem}},
	}

	for _, test := range tests {
		t.Run(test.instance, func(t *testing.T) {
			got := NewPlayerName(test.instance, test.source)
			if got.Name != test.expected.Name {
				t.Errorf("expected Name %q, got %q", test.expected.Name, got.Name)
			}
			if got.Instance != test.expected.Instance {
				t.Errorf("expected Instance %q, got %q", test.expected.Instance, got.Instance)
			}
			if got.Source != test.expected.Source {
				t.Errorf("expected Source %v, got %v", test.expected.Source, got.Source)
			}
		})
	}
}

func TestPlayerNameCompare(t *testing.T) {
	p1 := NewPlayerName("vlc.instance123", SourceDBusSession)
	p2 := NewPlayerName("vlc.instance123", SourceDBusSession)
	p3 := NewPlayerName("vlc.instance124", SourceDBusSession)
	p4 := NewPlayerName("spotify", SourceDBusSession)
	p5 := NewPlayerName("vlc.instance123", SourceDBusSystem)

	if p1.Compare(p2) != 0 {
		t.Errorf("expected p1 and p2 to be equal")
	}
	if p1.Compare(p3) == 0 {
		t.Errorf("expected p1 and p3 to be unequal")
	}
	if p1.Compare(p4) == 0 {
		t.Errorf("expected p1 and p4 to be unequal")
	}
	if p1.Compare(p5) == 0 {
		t.Errorf("expected p1 and p5 to be unequal (different source)")
	}
}

func TestPlayerNameInstanceCompare(t *testing.T) {
	p1 := NewPlayerName("vlc.instance123", SourceDBusSession)
	p2 := NewPlayerName("vlc", SourceDBusSession) // Name prefix match
	p3 := NewPlayerName("%any", SourceDBusSession)

	if p1.InstanceCompare(p2) != 0 {
		t.Errorf("expected p1 and p2 to be equal as instances")
	}
	if p2.InstanceCompare(p1) != 0 {
		t.Errorf("expected p2 and p1 to be equal as instances")
	}
	if p1.InstanceCompare(p3) != 0 {
		t.Errorf("expected p1 and any to be equal")
	}
}

func TestStringInstanceCompare(t *testing.T) {
	tests := []struct {
		name     string
		instance string
		expected int
	}{
		{"vlc", "vlc.instance123", 0},
		{"vlc.instance123", "vlc", 0},
		{"vlc", "vlc", 0},
		{"vlc", "spotify", 1},
		{"%any", "vlc", 0},
		{"vlc", "%any", 0},
		{"spotify", "spotifyd", 1}, // spotify should not match spotifyd
	}

	for _, test := range tests {
		if got := StringInstanceCompare(test.name, test.instance); got != test.expected {
			t.Errorf("StringInstanceCompare(%q, %q) = %d, want %d", test.name, test.instance, got, test.expected)
		}
	}
}
