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


func TestPlayerManagerFilterPlayerNamesIgnored(t *testing.T) {
	s1 := "selection1"
	s1i := "selection1.i_123"
	s2 := "selection2"
	s3 := "selection3"
	m4 := "selection4"
	m5 := "selection5"
	s6i := "selection6.i_2"
	anyPlayer := "%any"

	allPlayers := []string{
		"org.mpris.MediaPlayer2." + s1,
		"org.mpris.MediaPlayer2." + s1i,
		"org.mpris.MediaPlayer2." + s2,
		"org.mpris.MediaPlayer2." + s3,
		"org.mpris.MediaPlayer2." + s6i,
	}

	orig := listNamesOnBus
	defer func() { listNamesOnBus = orig }()
	listNamesOnBus = func(source Source) ([]string, error) {
		return allPlayers, nil
	}

	m, err := NewPlayerManager(SourceDBusSession)
	if err != nil {
		t.Fatalf("NewPlayerManager error: %v", err)
	}

	tests := []struct {
		allow    []string
		ignore   []string
		expected []string
	}{
		{[]string{s1}, []string{s1}, []string{s1i}},
		{[]string{s3, s1}, []string{s3}, []string{s1, s1i}},
		{[]string{s2, s1, s3}, []string{s1, s3}, []string{s1i, s2}},
		{[]string{s1, s2}, []string{s2}, []string{s1, s1i}},
		{[]string{m4, m5, s2, s3}, []string{s2}, []string{s3}},
		{[]string{m5, s1, m4, s3}, []string{s1, s3}, []string{s1i}},
		{[]string{anyPlayer}, []string{s1}, []string{s1i, s2, s3, s6i}},
		{[]string{s1, anyPlayer}, []string{s1}, []string{s1i, s2, s3, s6i}},
		{[]string{anyPlayer, s1}, []string{s1}, []string{s1i, s2, s3, s6i}},
	}

	for _, tt := range tests {
		filtered := m.FilterPlayerNames(tt.allow, tt.ignore)
		var got []string
		for _, pn := range filtered {
			got = append(got, pn.Instance)
		}

		if len(got) != len(tt.expected) {
			t.Errorf("FilterPlayerNames(%v, %v) length got %d, want %d", tt.allow, tt.ignore, len(got), len(tt.expected))
		}

		for i := 0; i < len(got) && i < len(tt.expected); i++ {
			if got[i] != tt.expected[i] {
				t.Errorf("FilterPlayerNames(%v, %v)[%d] got %s, want %s", tt.allow, tt.ignore, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestPlayerManagerFilterPlayerNamesSelections(t *testing.T) {
	s1 := "selection1"
	s1i := "selection1.i_123"
	s2 := "selection2"
	s3 := "selection3"
	m4 := "selection4"
	m5 := "selection5"
	m6 := "selection6"
	s6i := "selection6.i_2"
	anyPlayer := "%any"

	allPlayers := []string{
		"org.mpris.MediaPlayer2." + s1,
		"org.mpris.MediaPlayer2." + s1i,
		"org.mpris.MediaPlayer2." + s2,
		"org.mpris.MediaPlayer2." + s3,
		"org.mpris.MediaPlayer2." + s6i,
	}

	orig := listNamesOnBus
	defer func() { listNamesOnBus = orig }()
	listNamesOnBus = func(source Source) ([]string, error) {
		return allPlayers, nil
	}

	m, err := NewPlayerManager(SourceDBusSession)
	if err != nil {
		t.Fatalf("NewPlayerManager error: %v", err)
	}

	tests := []struct {
		allow    []string
		expected []string
	}{
		{[]string{s1}, []string{s1, s1i}},
		{[]string{s3, s1}, []string{s1, s1i, s3}},
		{[]string{s2, s1, s3}, []string{s1, s1i, s2, s3}},
		{[]string{s1, s2}, []string{s1, s1i, s2}},
		{[]string{m4, m5, s2, s3}, []string{s2, s3}},
		{[]string{m5, s1, m4, s3}, []string{s1, s1i, s3}},
		{[]string{s1, s1i}, []string{s1, s1i}},
		{[]string{s1i, s1}, []string{s1, s1i}},
		{[]string{m6, s1}, []string{s1, s1i, s6i}},
		{[]string{m4, m6, s3}, []string{s3, s6i}},
		{[]string{anyPlayer}, []string{s1, s1i, s2, s3, s6i}},
		{[]string{s1, anyPlayer}, []string{s1, s1i, s2, s3, s6i}},
		{[]string{anyPlayer, s1}, []string{s1, s1i, s2, s3, s6i}},
		{[]string{m6, anyPlayer, s2}, []string{s1, s1i, s2, s3, s6i}},
		{[]string{m6, s1, anyPlayer, s2}, []string{s1, s1i, s2, s3, s6i}},
	}

	for _, tt := range tests {
		filtered := m.FilterPlayerNames(tt.allow, nil)
		var got []string
		for _, pn := range filtered {
			got = append(got, pn.Instance)
		}

		if len(got) != len(tt.expected) {
			t.Errorf("FilterPlayerNames(%v, nil) length got %d, want %d", tt.allow, len(got), len(tt.expected))
			continue
		}

		for i := 0; i < len(got); i++ {
			if got[i] != tt.expected[i] {
				t.Errorf("FilterPlayerNames(%v, nil)[%d] got %s, want %s", tt.allow, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestPlayerManagerFilterPlayerNamesDaemonSelection(t *testing.T) {
	s1 := "selection1"
	s1i := "selection1.i_123"
	s2 := "selection2"
	s2i := "selection2.i_123"
	s3 := "selection3"
	s6i := "selection6.i_2"
	anyPlayer := "%any"

	allPlayers := []string{s1, s1i, s2, s3, s6i}

	tests := []struct {
		allow       []string
		daemonNames []string
		expected    []string
	}{
		{nil, allPlayers, allPlayers},
		{allPlayers, allPlayers, allPlayers},
		{[]string{s2}, []string{s1, s2}, []string{s2}},
		{[]string{s1}, []string{s2, s1i, s1}, []string{s1i, s1}},
		{[]string{s1}, []string{s2, s1, s1i}, []string{s1, s1i}},
		{[]string{s1i, s1}, []string{s1, s1i}, []string{s1, s1i}},
		{[]string{anyPlayer}, allPlayers, allPlayers},
		{[]string{anyPlayer, s1}, []string{s1, s1i, s2i, s2}, []string{s1, s1i, s2i, s2}},
		{[]string{anyPlayer, s1}, []string{s1, s1i, s2, s2i}, []string{s1, s1i, s2, s2i}},
		{[]string{anyPlayer, s1}, []string{s1i, s1, s2i, s2}, []string{s1i, s1, s2i, s2}},
		{[]string{anyPlayer, s1}, []string{s1i, s1, s2, s2i}, []string{s1i, s1, s2, s2i}},
		{[]string{s2, anyPlayer}, []string{s1, s1i, s2i, s2}, []string{s1, s1i, s2i, s2}},
		{[]string{s2, anyPlayer}, []string{s1, s1i, s2, s2i}, []string{s1, s1i, s2, s2i}},
		{[]string{s2, anyPlayer}, []string{s1i, s1, s2i, s2}, []string{s1i, s1, s2i, s2}},
		{[]string{s2, anyPlayer}, []string{s1i, s1, s2, s2i}, []string{s1i, s1, s2, s2i}},
		{[]string{s2i, anyPlayer}, []string{s1i, s1, s2, s2i}, []string{s1i, s1, s2, s2i}},
	}

	for _, tt := range tests {
		func() {
			orig := listNamesOnBus
			defer func() { listNamesOnBus = orig }()
			listNamesOnBus = func(source Source) ([]string, error) {
				names := []string{"org.mpris.MediaPlayer2.playerctld"}
				return names, nil
			}

			m, err := NewPlayerManager(SourceDBusSession)
			if err != nil {
				t.Fatalf("NewPlayerManager error: %v", err)
			}

			m.playerNames = make([]*PlayerName, 0)
			for _, n := range tt.daemonNames {
				pn := &PlayerName{Name: n, Instance: n, Source: SourceDBusSession}
				m.playerNames = append(m.playerNames, pn)
			}

			filtered := m.FilterPlayerNames(tt.allow, nil)
			var got []string
			for _, pn := range filtered {
				got = append(got, pn.Instance)
			}

			if len(got) != len(tt.expected) {
				t.Errorf("FilterPlayerNames(%v) daemon names %v length got %d, want %d", tt.allow, tt.daemonNames, len(got), len(tt.expected))
				return
			}

			for i := 0; i < len(got); i++ {
				if got[i] != tt.expected[i] {
					t.Errorf("FilterPlayerNames(%v) daemon names %v [%d] got %s, want %s", tt.allow, tt.daemonNames, i, got[i], tt.expected[i])
				}
			}
		}()
	}
}
