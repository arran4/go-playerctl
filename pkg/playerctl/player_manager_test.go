package playerctl

import "testing"

func TestPlayerManagerRefreshAndFilter(t *testing.T) {
	orig := listNamesOnBus
	defer func() { listNamesOnBus = orig }()

	listNamesOnBus = func(source Source) ([]string, error) {
		if source == SourceDBusSession {
			return []string{"org.mpris.MediaPlayer2.vlc", "org.mpris.MediaPlayer2.spotify.instanceA", "org.mpris.MediaPlayer2.playerctld"}, nil
		}
		return []string{"org.mpris.MediaPlayer2.mpd"}, nil
	}

	m, err := NewPlayerManager(SourceNone)
	if err != nil {
		t.Fatalf("NewPlayerManager error: %v", err)
	}
	if len(m.PlayerNames()) != 3 {
		t.Fatalf("expected 3 names, got %d", len(m.PlayerNames()))
	}

	filtered := m.FilterPlayerNames([]string{"spotify", "mpd"}, []string{"mpd"})
	if len(filtered) != 1 || filtered[0].Name != "spotify" {
		t.Fatalf("unexpected filtered set: %+v", filtered)
	}
}

func TestPlayerManagerNameOwnerChangedAndMoveTop(t *testing.T) {
	orig := listNamesOnBus
	defer func() { listNamesOnBus = orig }()
	listNamesOnBus = func(source Source) ([]string, error) { return nil, nil }

	m, err := NewPlayerManager(SourceDBusSession)
	if err != nil {
		t.Fatalf("NewPlayerManager error: %v", err)
	}

	m.HandleNameOwnerChanged("org.mpris.MediaPlayer2.vlc", "", ":1.20", SourceDBusSession)
	if len(m.PlayerNames()) != 1 {
		t.Fatalf("expected name add, got %d", len(m.PlayerNames()))
	}

	p1, _ := NewPlayer("a.one", SourceNone)
	p2, _ := NewPlayer("b.two", SourceNone)
	m.ManagePlayer(p1)
	m.ManagePlayer(p2)
	m.MovePlayerToTop(p2)
	if m.Players()[0] != p2 {
		t.Fatal("expected p2 at top")
	}

	m.HandleNameOwnerChanged("org.mpris.MediaPlayer2.vlc", ":1.20", "", SourceDBusSession)
	if len(m.PlayerNames()) != 0 {
		t.Fatalf("expected name remove, got %d", len(m.PlayerNames()))
	}
}
