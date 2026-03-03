package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/arran4/go-playerctl/pkg/playerctl"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tuiModel struct {
	players  []string
	cursor   int
	status   string
	metadata string
	err      error
}

func initialModel(instances []string) tuiModel {
	m := tuiModel{
		players: instances,
	}
	m.updateCurrentPlayerInfo()
	return m
}

func (m *tuiModel) updateCurrentPlayerInfo() {
	if len(m.players) == 0 {
		m.status = "No players found"
		m.metadata = ""
		return
	}

	if m.cursor >= len(m.players) {
		m.cursor = len(m.players) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	instance := m.players[m.cursor]
	p, err := newPlayer(instance, playerctl.SourceDBusSession)
	if err != nil {
		m.status = "Error connecting"
		m.metadata = err.Error()
		return
	}
	defer p.Close()

	status, err := p.PlaybackStatus()
	if err != nil {
		m.status = "Unknown"
	} else {
		m.status = status.String()
	}

	title, _ := p.GetTitle()
	artist, _ := p.GetArtist()
	album, _ := p.GetAlbum()

	parts := []string{}
	if title != "" {
		parts = append(parts, fmt.Sprintf("Title: %s", title))
	}
	if artist != "" {
		parts = append(parts, fmt.Sprintf("Artist: %s", artist))
	}
	if album != "" {
		parts = append(parts, fmt.Sprintf("Album: %s", album))
	}
	m.metadata = strings.Join(parts, "\n")
	if m.metadata == "" {
		m.metadata = "No metadata"
	}
}

func (m *tuiModel) refreshPlayers() {
	manager, err := newPlayerManger(playerctl.SourceNone)
	if err == nil {
		names := manager.PlayerNames()
		newPlayers := make([]string, len(names))
		for i, n := range names {
			newPlayers[i] = n.Instance
		}

		var currentInstance string
		if len(m.players) > 0 && m.cursor >= 0 && m.cursor < len(m.players) {
		    currentInstance = m.players[m.cursor]
		}

		m.players = newPlayers

		// Try to keep cursor on the same instance
		found := false
		if currentInstance != "" {
            for i, p := range m.players {
                if p == currentInstance {
                    m.cursor = i
                    found = true
                    break
                }
            }
		}
		if !found {
		    m.cursor = 0
		}
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tickCmd()
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateCurrentPlayerInfo()
			}
		case "down", "j":
			if m.cursor < len(m.players)-1 {
				m.cursor++
				m.updateCurrentPlayerInfo()
			}
		case " ": // Play/Pause
			if len(m.players) > 0 {
				p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
				if err == nil {
					p.PlayPause()
					p.Close()
					m.updateCurrentPlayerInfo()
				}
			}
		case "right", "l": // Next
			if len(m.players) > 0 {
				p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
				if err == nil {
					p.Next()
					p.Close()
					m.updateCurrentPlayerInfo()
				}
			}
		case "left", "h": // Previous
			if len(m.players) > 0 {
				p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
				if err == nil {
					p.Previous()
					p.Close()
					m.updateCurrentPlayerInfo()
				}
			}
		case "r": // Refresh players manually
		    m.refreshPlayers()
		    m.updateCurrentPlayerInfo()
		}
	case tickMsg:
		// Instead of synchronously updating, we could return a tea.Cmd here if it was a complex app.
		// For now, since this is a relatively fast D-Bus call to local services, it's tolerable in the main loop.
		// However, returning it as a command would be better. We'll do a simple synchronous update here for simplicity.
		m.updateCurrentPlayerInfo()
		return m, tickCmd()
	}
	return m, nil
}

func (m tuiModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	b.WriteString("Media Players (use ↑/↓ to navigate, space to play/pause, ←/→ for prev/next, r to refresh):\n\n")

	for i, player := range m.players {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(fmt.Sprintf("%s %s\n", cursor, player)))
		} else {
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, player))
		}
	}

	b.WriteString("\n")

	if len(m.players) > 0 {
		b.WriteString(lipgloss.NewStyle().Underline(true).Render("Status:"))
		b.WriteString(fmt.Sprintf(" %s\n\n", m.status))

		b.WriteString(lipgloss.NewStyle().Underline(true).Render("Metadata:"))
		b.WriteString("\n")
		b.WriteString(m.metadata)
		b.WriteString("\n")
	} else {
		b.WriteString("No players available.\n")
	}

	b.WriteString("\nPress 'q' to quit.\n")

	return b.String()
}

func runTUI(instances []string, stdout, stderr io.Writer, opts cliOptions) int {
    if len(instances) == 0 {
        // If instances is empty, it means we probably need to fetch all players
        manager, err := newPlayerManger(playerctl.SourceNone)
        if err != nil {
            fmt.Fprintf(stderr, "failed to get players: %v\n", err)
            return 1
        }
        for _, n := range manager.PlayerNames() {
            instances = append(instances, n.Instance)
        }
    }
	p := tea.NewProgram(initialModel(instances), tea.WithOutput(stdout))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(stderr, "Error running TUI: %v\n", err)
		return 1
	}
	return 0
}
