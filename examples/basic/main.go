package main

import (
	"fmt"
	"os"
	"time"

	"github.com/arran4/go-playerctl/pkg/playerctl"
)

func main() {
	// Initialize a player manager to discover players
	manager, err := playerctl.NewPlayerManager(playerctl.SourceNone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating manager: %v\n", err)
		os.Exit(1)
	}

	names := manager.PlayerNames()
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "No players found")
		os.Exit(1)
	}

	// Connect to the first available player
	player, err := playerctl.NewPlayerFromName(names[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating player: %v\n", err)
		os.Exit(1)
	}
	defer player.Close()

	// Initial check for Lana Del Rey
	artist, err := player.GetArtist()
	if err == nil && artist == "Lana Del Rey" {
		player.Next()
	}

	// Start playing some music
	player.Play()

	var lastArtist, lastTitle string
	var lastStatus playerctl.PlaybackStatus
	lastVolume := -1.0

	// Emulate events using a polling loop
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	fmt.Println("Waiting for events...")
	for range ticker.C {
		if player.Disappeared() {
			fmt.Println("Player disappeared")
			break
		}

		// Check metadata
		currentArtist, _ := player.GetArtist()
		currentTitle, _ := player.GetTitle()

		if currentArtist != lastArtist || currentTitle != lastTitle {
			if currentArtist != "" && currentTitle != "" {
				fmt.Println("Now playing:")
				fmt.Printf("%s - %s\n", currentArtist, currentTitle)
			}
			lastArtist = currentArtist
			lastTitle = currentTitle
		}

		// Check playback status
		status, err := player.PlaybackStatus()
		if err == nil && status != lastStatus {
			if status == playerctl.PlaybackStatusPlaying {
				vol, _ := player.Volume()
				if vol != lastVolume {
					fmt.Printf("Playing at volume %v\n", vol)
					lastVolume = vol
				} else {
					fmt.Println("Playing")
				}
			} else if status == playerctl.PlaybackStatusPaused {
				title, _ := player.GetTitle()
				fmt.Printf("Paused the song: %s\n", title)
			}
			lastStatus = status
		}
	}
}
