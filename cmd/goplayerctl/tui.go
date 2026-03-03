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
	players       []string
	cursor        int
	status        string
	metadata      string
	controlScheme string
	position      int64
	length        int64
	err           error
}

var controlSchemes = []string{"arrow", "vim", "winamp", "emacs"}

func initialModel(instances []string) tuiModel {
	m := tuiModel{
		players:       instances,
		controlScheme: "arrow",
	}
	m.updateCurrentPlayerInfo()
	return m
}

func (m *tuiModel) cycleControlScheme() {
	for i, scheme := range controlSchemes {
		if scheme == m.controlScheme {
			next := (i + 1) % len(controlSchemes)
			m.controlScheme = controlSchemes[next]
			return
		}
	}
	m.controlScheme = "arrow"
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

	pos, err := p.Position()
	if err == nil {
		m.position = pos
	} else {
		m.position = 0
	}

	m.length = 0
	meta, err := p.Metadata()
	if err == nil {
		if v, ok := meta["mpris:length"]; ok {
			switch t := v.Value().(type) {
			case int64:
				m.length = t
			case uint64:
				m.length = int64(t)
			case int32:
				m.length = int64(t)
			case float64:
				m.length = int64(t)
			}
		}
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
		key := msg.String()
		switch key {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			m.refreshPlayers()
			m.updateCurrentPlayerInfo()
		case "tab", "s":
			m.cycleControlScheme()
		default:
			action := m.mapKeyEvent(key)
			m.handleAction(action)
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

type tuiAction string

const (
	actionUp        tuiAction = "up"
	actionDown      tuiAction = "down"
	actionPlayPause tuiAction = "playpause"
	actionPause     tuiAction = "pause"
	actionStop      tuiAction = "stop"
	actionNext      tuiAction = "next"
	actionPrev      tuiAction = "prev"
	actionNone      tuiAction = "none"
)

func (m *tuiModel) mapKeyEvent(key string) tuiAction {
	switch m.controlScheme {
	case "arrow":
		switch key {
		case "up":
			return actionUp
		case "down":
			return actionDown
		case " ":
			return actionPlayPause
		case "left":
			return actionPrev
		case "right":
			return actionNext
		}
	case "vim":
		switch key {
		case "k":
			return actionUp
		case "j":
			return actionDown
		case " ":
			return actionPlayPause
		case "h":
			return actionPrev
		case "l":
			return actionNext
		}
	case "winamp":
		switch key {
		case "up":
			return actionUp
		case "down":
			return actionDown
		case "z":
			return actionPrev
		case "x":
			return actionPlayPause
		case "c":
			return actionPause
		case "v":
			return actionStop
		case "b":
			return actionNext
		}
	case "emacs":
		switch key {
		case "p":
			return actionUp
		case "n":
			return actionDown
		case " ":
			return actionPlayPause
		case "b":
			return actionPrev
		case "f":
			return actionNext
		}
	}
	return actionNone
}

func (m *tuiModel) handleAction(action tuiAction) {
	switch action {
	case actionUp:
		if m.cursor > 0 {
			m.cursor--
			m.updateCurrentPlayerInfo()
		}
	case actionDown:
		if m.cursor < len(m.players)-1 {
			m.cursor++
			m.updateCurrentPlayerInfo()
		}
	case actionPlayPause:
		if len(m.players) > 0 {
			p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
			if err == nil {
				p.PlayPause()
				p.Close()
				m.updateCurrentPlayerInfo()
			}
		}
	case actionPause:
		if len(m.players) > 0 {
			p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
			if err == nil {
				p.Pause()
				p.Close()
				m.updateCurrentPlayerInfo()
			}
		}
	case actionStop:
		if len(m.players) > 0 {
			p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
			if err == nil {
				p.Stop()
				p.Close()
				m.updateCurrentPlayerInfo()
			}
		}
	case actionNext:
		if len(m.players) > 0 {
			p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
			if err == nil {
				p.Next()
				p.Close()
				m.updateCurrentPlayerInfo()
			}
		}
	case actionPrev:
		if len(m.players) > 0 {
			p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
			if err == nil {
				p.Previous()
				p.Close()
				m.updateCurrentPlayerInfo()
			}
		}
	}
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(0, 1).
			Width(40)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#01FAC6")).
				Bold(true)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8A8A8A"))
)

func (m tuiModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Go Playerctl TUI"))
	b.WriteString(fmt.Sprintf(" [Scheme: %s (press tab to change)]\n\n", m.controlScheme))

	playersBox := ""
	for i, player := range m.players {
		if m.cursor == i {
			playersBox += selectedItemStyle.Render("> "+player) + "\n"
		} else {
			playersBox += itemStyle.Render("  "+player) + "\n"
		}
	}
	if len(m.players) == 0 {
		playersBox = "No players available.\n"
	}
	playersBoxStr := boxStyle.Render(playersBox)

	metaBox := ""
	if len(m.players) > 0 {
		metaBox += lipgloss.NewStyle().Underline(true).Render("Status:") + " " + m.status + "\n\n"
		metaBox += lipgloss.NewStyle().Underline(true).Render("Metadata:") + "\n" + m.metadata + "\n"

		if m.length > 0 && m.status != "Stopped" {
			metaBox += "\n"
			width := 36
			filled := int((float64(m.position) / float64(m.length)) * float64(width))
			if filled > width {
				filled = width
			} else if filled < 0 {
				filled = 0
			}
			empty := width - filled

			bar := lipgloss.NewStyle().Foreground(lipgloss.Color("#01FAC6")).Render(strings.Repeat("█", filled)) +
				lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("░", empty))

			posSec := time.Duration(m.position/1000) * time.Millisecond
			lenSec := time.Duration(m.length/1000) * time.Millisecond

			metaBox += fmt.Sprintf("%s\n%s / %s", bar, posSec.Round(time.Second), lenSec.Round(time.Second))
		}
	} else {
		metaBox = "No metadata\n"
	}
	metaBoxStr := boxStyle.Render(metaBox)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, playersBoxStr, metaBoxStr))
	b.WriteString("\n\n")

	keysHelp := ""
	switch m.controlScheme {
	case "arrow":
		keysHelp = "↑/↓: navigate • ←/→: prev/next • space: play/pause"
	case "vim":
		keysHelp = "k/j: navigate • h/l: prev/next • space: play/pause"
	case "winamp":
		keysHelp = "↑/↓: navigate • z/b: prev/next • x/c/v: play/pause/stop"
	case "emacs":
		keysHelp = "p/n: navigate • b/f: prev/next • space: play/pause"
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(keysHelp + " • r: refresh • q: quit"))
	b.WriteString("\n")

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
