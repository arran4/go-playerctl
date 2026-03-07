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
	volume        float64
	jumpSizeIndex int
	err           error
	width         int
	height        int
}

var controlSchemes = []string{"arrow", "vim", "winamp", "emacs"}
var jumpSizes = []int64{5_000_000, 10_000_000, 30_000_000, 60_000_000} // microseconds

func (m *tuiModel) cycleJumpSize() {
	m.jumpSizeIndex = (m.jumpSizeIndex + 1) % len(jumpSizes)
}

func initialModel(instances []string, defaultScheme string) tuiModel {
	scheme := "arrow"
	for _, s := range controlSchemes {
		if s == defaultScheme {
			scheme = s
			break
		}
	}

	m := tuiModel{
		players:       instances,
		controlScheme: scheme,
		volume:        -1.0,
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

	vol, err := p.Volume()
	if err == nil {
		m.volume = vol
	} else {
		m.volume = -1.0
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			m.refreshPlayers()
			m.updateCurrentPlayerInfo()
		case "alt+s":
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
	actionUp           tuiAction = "up"
	actionDown         tuiAction = "down"
	actionPlayPause    tuiAction = "playpause"
	actionPause        tuiAction = "pause"
	actionStop         tuiAction = "stop"
	actionNext         tuiAction = "next"
	actionPrev         tuiAction = "prev"
	actionNone         tuiAction = "none"
	actionVolumeUp     tuiAction = "volume_up"
	actionVolumeDown   tuiAction = "volume_down"
	actionCycleJump    tuiAction = "cycle_jump"
	actionSeekForward  tuiAction = "seek_forward"
	actionSeekBackward tuiAction = "seek_backward"
)

func (m *tuiModel) mapKeyEvent(key string) tuiAction {
	switch key {
	case "+", "=":
		return actionVolumeUp
	case "-", "_":
		return actionVolumeDown
	case "s":
		return actionCycleJump
	case "[", "<", "pgup":
		return actionPrev
	case "]", ">", "pgdown":
		return actionNext
	}

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
			return actionSeekBackward
		case "right":
			return actionSeekForward
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
			return actionSeekBackward
		case "l":
			return actionSeekForward
		}
	case "winamp":
		switch key {
		case "up":
			return actionUp
		case "down":
			return actionDown
		case "left":
			return actionSeekBackward
		case "right":
			return actionSeekForward
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
			return actionSeekBackward
		case "f":
			return actionSeekForward
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
	case actionVolumeUp:
		if len(m.players) > 0 && m.volume >= 0 {
			p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
			if err == nil {
				newVol := m.volume + 0.05
				if newVol > 1.0 {
					newVol = 1.0
				}
				p.SetVolume(newVol)
				p.Close()
				m.updateCurrentPlayerInfo()
			}
		}
	case actionVolumeDown:
		if len(m.players) > 0 && m.volume >= 0 {
			p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
			if err == nil {
				newVol := m.volume - 0.05
				if newVol < 0.0 {
					newVol = 0.0
				}
				p.SetVolume(newVol)
				p.Close()
				m.updateCurrentPlayerInfo()
			}
		}
	case actionCycleJump:
		m.cycleJumpSize()
	case actionSeekForward:
		if len(m.players) > 0 {
			p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
			if err == nil {
				jumpAmt := jumpSizes[m.jumpSizeIndex]
				// Try to not seek past the end if we know the length
				if m.length > 0 && m.position+jumpAmt > m.length {
					trackId, _ := p.GetTrackID()
					if trackId != "" {
						p.SetPosition(trackId, m.length)
					}
				} else {
					p.Seek(jumpAmt)
				}
				p.Close()
				m.updateCurrentPlayerInfo()
			}
		}
	case actionSeekBackward:
		if len(m.players) > 0 {
			p, err := newPlayer(m.players[m.cursor], playerctl.SourceDBusSession)
			if err == nil {
				jumpAmt := jumpSizes[m.jumpSizeIndex]
				if m.position-jumpAmt < 0 {
					trackId, _ := p.GetTrackID()
					if trackId != "" {
						p.SetPosition(trackId, 0)
					}
				} else {
					p.Seek(-jumpAmt)
				}
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
			Padding(0, 1)

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
	jumpSecs := jumpSizes[m.jumpSizeIndex] / 1_000_000
	b.WriteString(fmt.Sprintf(" [Scheme: %s (alt+s)] [Jump: %ds (s)]\n\n", m.controlScheme, jumpSecs))

	boxWidth := 40
	if m.width > 0 {
		calculated := (m.width / 2) - 4
		if calculated > 40 {
			boxWidth = calculated
		}
	}
	currentBoxStyle := boxStyle.Width(boxWidth)

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
	playersBoxStr := currentBoxStyle.Render(playersBox)

	metaBox := ""
	if len(m.players) > 0 {
		metaBox += lipgloss.NewStyle().Underline(true).Render("Status:") + " " + m.status + "\n\n"
		metaBox += lipgloss.NewStyle().Underline(true).Render("Metadata:") + "\n" + m.metadata + "\n"

		if m.volume >= 0 {
			metaBox += fmt.Sprintf("Volume: %.0f%%\n", m.volume*100)
		}

		if m.length > 0 && m.status != "Stopped" {
			metaBox += "\n"
			progressBarWidth := boxWidth - 4
			if progressBarWidth < 10 {
				progressBarWidth = 10
			}
			filled := int((float64(m.position) / float64(m.length)) * float64(progressBarWidth))
			if filled > progressBarWidth {
				filled = progressBarWidth
			} else if filled < 0 {
				filled = 0
			}
			empty := progressBarWidth - filled

			bar := lipgloss.NewStyle().Foreground(lipgloss.Color("#01FAC6")).Render(strings.Repeat("█", filled)) +
				lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("░", empty))

			posSec := time.Duration(m.position/1000) * time.Millisecond
			lenSec := time.Duration(m.length/1000) * time.Millisecond

			metaBox += fmt.Sprintf("%s\n%s / %s", bar, posSec.Round(time.Second), lenSec.Round(time.Second))
		}
	} else {
		metaBox = "No metadata\n"
	}
	metaBoxStr := currentBoxStyle.Render(metaBox)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, playersBoxStr, metaBoxStr))
	b.WriteString("\n\n")

	keysHelp := ""
	switch m.controlScheme {
	case "arrow":
		keysHelp = "↑/↓: navigate • ←/→: seek • space: play/pause"
	case "vim":
		keysHelp = "k/j: navigate • h/l: seek • space: play/pause"
	case "winamp":
		keysHelp = "↑/↓: navigate • ←/→: seek • z/b: prev/next • x/c/v: play/pause/stop"
	case "emacs":
		keysHelp = "p/n: navigate • b/f: seek • space: play/pause"
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(keysHelp + " • [/]: prev/next track • +/-: vol • r: refresh • q: quit"))
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
	p := tea.NewProgram(initialModel(instances, opts.tuiScheme), tea.WithOutput(stdout))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(stderr, "Error running TUI: %v\n", err)
		return 1
	}
	return 0
}
