package playerctl

import (
	"sort"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

const dbusBusName = "org.freedesktop.DBus"

var listNamesOnBus = func(source Source) ([]string, error) {
	if source == SourceNone {
		return nil, nil
	}

	var (
		conn *dbus.Conn
		err  error
	)
	if source == SourceDBusSession {
		conn, err = connectSessionBus()
	} else {
		conn, err = connectSystemBus()
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var names []string
	call := conn.Object(dbusBusName, "/org/freedesktop/DBus").Call("org.freedesktop.DBus.ListNames", 0)
	if call.Err != nil {
		return nil, call.Err
	}
	if err := call.Store(&names); err != nil {
		return nil, err
	}
	return names, nil
}

// PlayerManager tracks discovered players and managed player instances.
type PlayerManager struct {
	mu sync.RWMutex

	source Source

	playerNames []*PlayerName
	players     []*Player
}

// NewPlayerManager constructs a manager for a given bus source.
func NewPlayerManager(source Source) (*PlayerManager, error) {
	m := &PlayerManager{source: source}
	if err := m.Refresh(); err != nil {
		return nil, err
	}
	return m, nil
}

// Source returns manager bus source.
func (m *PlayerManager) Source() Source {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.source
}

// PlayerNames returns a copy of discovered player names.
func (m *PlayerManager) PlayerNames() []*PlayerName {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*PlayerName, len(m.playerNames))
	copy(out, m.playerNames)
	return out
}

// Players returns a copy of managed players.
func (m *PlayerManager) Players() []*Player {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Player, len(m.players))
	copy(out, m.players)
	return out
}

// Refresh performs discovery via org.freedesktop.DBus ListNames.
func (m *PlayerManager) Refresh() error {
	sources := []Source{m.source}
	if m.source == SourceNone {
		sources = []Source{SourceDBusSession, SourceDBusSystem}
	}

	seen := make([]*PlayerName, 0)
	for _, source := range sources {
		names, err := listNamesOnBus(source)
		if err != nil {
			return err
		}
		for _, n := range names {
			if !strings.HasPrefix(n, "org.mpris.MediaPlayer2.") || n == "org.mpris.MediaPlayer2.playerctld" {
				continue
			}
			instance := strings.TrimPrefix(n, "org.mpris.MediaPlayer2.")
			seen = append(seen, NewPlayerName(instance, source))
		}
	}

	m.mu.Lock()
	m.playerNames = seen
	m.mu.Unlock()
	return nil
}

// HandleNameOwnerChanged updates manager state based on NameOwnerChanged signal args.
func (m *PlayerManager) HandleNameOwnerChanged(busName, oldOwner, newOwner string, source Source) {
	if !strings.HasPrefix(busName, "org.mpris.MediaPlayer2.") || busName == "org.mpris.MediaPlayer2.playerctld" {
		return
	}
	instance := strings.TrimPrefix(busName, "org.mpris.MediaPlayer2.")
	name := NewPlayerName(instance, source)

	m.mu.Lock()
	defer m.mu.Unlock()

	if oldOwner == "" && newOwner != "" {
		m.addNameLocked(name)
		return
	}
	if oldOwner != "" && newOwner == "" {
		m.removeNameLocked(name)
		m.removePlayerLocked(instance, source)
	}
}

func (m *PlayerManager) addNameLocked(pn *PlayerName) {
	for _, existing := range m.playerNames {
		if existing.Compare(pn) == 0 {
			return
		}
	}
	m.playerNames = append(m.playerNames, pn)
}

func (m *PlayerManager) removeNameLocked(target *PlayerName) {
	out := m.playerNames[:0]
	for _, pn := range m.playerNames {
		if pn.Compare(target) != 0 {
			out = append(out, pn)
		}
	}
	m.playerNames = out
}

func (m *PlayerManager) removePlayerLocked(instance string, source Source) {
	out := m.players[:0]
	for _, p := range m.players {
		if p.Instance() == instance && p.Source() == source {
			continue
		}
		out = append(out, p)
	}
	m.players = out
}

// ManagePlayer adds a connected player to the managed list.
func (m *PlayerManager) ManagePlayer(player *Player) {
	if player == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.players = append(m.players, player)
}

// MovePlayerToTop moves a managed player to the front, preserving order otherwise.
func (m *PlayerManager) MovePlayerToTop(player *Player) {
	if player == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, p := range m.players {
		if p == player {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return
	}
	m.players = append([]*Player{player}, append(m.players[:idx], m.players[idx+1:]...)...)
}

// FilterPlayerNames applies --player/%any style selection and ignore lists.
func (m *PlayerManager) FilterPlayerNames(allow []string, ignore []string) []*PlayerName {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ignoreSet := map[string]struct{}{}
	for _, i := range ignore {
		ignoreSet[i] = struct{}{}
	}

	out := make([]*PlayerName, 0, len(m.playerNames))
	for _, pn := range m.playerNames {
		if _, skip := ignoreSet[pn.Instance]; skip {
			continue
		}
		if len(allow) == 0 {
			out = append(out, pn)
			continue
		}
		for _, wanted := range allow {
			if StringInstanceCompare(wanted, pn.Instance) == 0 {
				out = append(out, pn)
				break
			}
		}
	}
	return out
}

// SortPlayersByActivity applies stable ordering callback. Lower weight ranks first.
func (m *PlayerManager) SortPlayersByActivity(weight func(*Player) int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sort.SliceStable(m.players, func(i, j int) bool {
		return weight(m.players[i]) < weight(m.players[j])
	})
}
