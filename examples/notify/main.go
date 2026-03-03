package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/arran4/go-playerctl/pkg/playerctl"
)

type PlayerState struct {
	Artist string
	Title  string
	Status playerctl.PlaybackStatus
}

func main() {
	manager, err := playerctl.NewPlayerManager(playerctl.SourceNone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating manager: %v\n", err)
		os.Exit(1)
	}

	knownPlayers := make(map[string]*playerctl.Player)
	playerStates := make(map[string]PlayerState)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	fmt.Println("Waiting for events to notify...")
	for range ticker.C {
		if err := manager.Refresh(); err != nil {
			fmt.Fprintf(os.Stderr, "Error refreshing manager: %v\n", err)
			continue
		}

		currentNames := make(map[string]bool)
		for _, pn := range manager.PlayerNames() {
			currentNames[pn.Instance] = true

			if _, exists := knownPlayers[pn.Instance]; !exists {
				player, err := playerctl.NewPlayerFromName(pn)
				if err == nil {
					knownPlayers[pn.Instance] = player
					playerStates[pn.Instance] = PlayerState{}
					manager.ManagePlayer(player)
					// Event: name-appeared -> notify initial state
					notify(player)
				}
			}
		}

		for instance, p := range knownPlayers {
			if !currentNames[instance] || p.Disappeared() {
				p.Close()
				delete(knownPlayers, instance)
				delete(playerStates, instance)
				continue
			}

			state := playerStates[instance]

			currentArtist, _ := p.GetArtist()
			currentTitle, _ := p.GetTitle()

			if currentArtist != state.Artist || currentTitle != state.Title {
				state.Artist = currentArtist
				state.Title = currentTitle
				notify(p)
			}

			status, err := p.PlaybackStatus()
			if err == nil && status != state.Status {
				if status == playerctl.PlaybackStatusPlaying {
					notify(p)
				}
				state.Status = status
			}

			playerStates[instance] = state
		}
	}
}

func notify(player *playerctl.Player) {
	artist, _ := player.GetArtist()
	title, _ := player.GetTitle()

	if artist == "" || title == "" {
		return
	}

	meta, err := player.Metadata()
	if err != nil {
		return
	}

	var iconPath string
	if v, ok := meta["xesam:url"]; ok {
		if ustr, ok := v.Value().(string); ok {
			if parsedURL, err := url.Parse(ustr); err == nil && parsedURL.Scheme == "file" {
				dir := filepath.Dir(parsedURL.Path)
				coverPath := filepath.Join(dir, "cover.jpg")
				if _, err := os.Stat(coverPath); err == nil {
					iconPath = coverPath
				}
			}
		}
	}

	args := []string{title, artist}
	if iconPath != "" {
		args = append(args, "-i", iconPath)
	}

	cmd := exec.Command("notify-send", args...)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending notification: %v\n", err)
	}
}
