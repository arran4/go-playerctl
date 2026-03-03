package main

import (
	"fmt"
	"os"
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

	fmt.Println("Waiting for events...")
	for range ticker.C {
		// Discover new players
		if err := manager.Refresh(); err != nil {
			fmt.Fprintf(os.Stderr, "Error refreshing manager: %v\n", err)
			continue
		}

		currentNames := make(map[string]bool)
		for _, pn := range manager.PlayerNames() {
			if pn.Name == "vlc" || pn.Name == "cmus" {
				currentNames[pn.Instance] = true

				if _, exists := knownPlayers[pn.Instance]; !exists {
					player, err := playerctl.NewPlayerFromName(pn)
					if err == nil {
						knownPlayers[pn.Instance] = player
						playerStates[pn.Instance] = PlayerState{}
						manager.ManagePlayer(player)
						// Event: name-appeared
						fmt.Printf("player has appeared: %s\n", pn.Name)
					}
				}
			}
		}

		// Check for vanished players and updates
		for instance, p := range knownPlayers {
			if !currentNames[instance] || p.Disappeared() {
				// Event: player-vanished
				fmt.Printf("player has exited: %s\n", p.Name())
				p.Close()
				delete(knownPlayers, instance)
				delete(playerStates, instance)
				continue
			}

			state := playerStates[instance]

			// Check metadata
			currentArtist, _ := p.GetArtist()
			currentTitle, _ := p.GetTitle()

			if currentArtist != state.Artist || currentTitle != state.Title {
				if currentArtist != "" && currentTitle != "" {
					fmt.Printf("%s - %s\n", currentArtist, currentTitle)
				}
				state.Artist = currentArtist
				state.Title = currentTitle
			}

			// Check playback status
			status, err := p.PlaybackStatus()
			if err == nil && status != state.Status {
				if status == playerctl.PlaybackStatusPlaying {
					fmt.Printf("player is playing: %s\n", p.Name())
				}
				state.Status = status
			}

			playerStates[instance] = state
		}
	}
}
