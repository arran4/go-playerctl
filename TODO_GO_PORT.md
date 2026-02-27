# Playerctl Go Port Todo List

This document outlines the tasks required to port the `playerctl` codebase from C to Go.

## Project Setup
- [ ] Initialize Go module (`go mod init github.com/altdesktop/playerctl`)
- [ ] Set up directory structure (e.g., `cmd/playerctl`, `cmd/playerctld`, `pkg/playerctl`)
- [ ] Choose a DBus library (e.g., `github.com/godbus/dbus/v5`)

## Core Library (`pkg/playerctl`)

### Enums and Types (`playerctl-common.h`, `playerctl-enum-types.h`)
- [ ] Define `PlayerctlPlaybackStatus` enum (Playing, Paused, Stopped)
- [ ] Define `PlayerctlLoopStatus` enum (None, Track, Playlist)
- [ ] Define `PlayerctlSource` enum (DBusSession, DBusSystem)
- [ ] Implement string conversion functions for enums

### Player Name Handling (`playerctl-player-name.c`)
- [ ] Define `PlayerName` struct (name, instance, source)
- [ ] Implement `PlayerName` comparison functions
- [ ] Implement `PlayerName` parsing/construction logic

### Player Implementation (`playerctl-player.c`)
- [ ] Define `Player` struct
- [ ] Implement constructor `NewPlayer` (connects to DBus)
- [ ] Implement `NewPlayerFromName`
- [ ] Implement DBus signal handling (`PropertiesChanged`, `Seeked`, `NameOwnerChanged`)
- [ ] Implement Property Getters:
    - [ ] `PlaybackStatus`
    - [ ] `LoopStatus`
    - [ ] `Shuffle`
    - [ ] `Volume`
    - [ ] `Position` (handle monotonic clock calculation)
    - [ ] `Metadata`
    - [ ] `CanControl`, `CanPlay`, `CanPause`, `CanSeek`, `CanGoNext`, `CanGoPrevious`
- [ ] Implement Commands:
    - [ ] `Play`, `Pause`, `PlayPause`, `Stop`
    - [ ] `Next`, `Previous`
    - [ ] `Seek`, `SetPosition`
    - [ ] `OpenUri`
    - [ ] `SetVolume`
    - [ ] `SetLoopStatus`
    - [ ] `SetShuffle`
- [ ] Implement Metadata parsing and helpers (`GetArtist`, `GetTitle`, `GetAlbum`, `GetTrackID`)

### Player Manager (`playerctl-player-manager.c`)
- [ ] Define `PlayerManager` struct
- [ ] Implement player discovery (ListNames on DBus)
- [ ] Implement monitoring for new/removed players (`NameOwnerChanged`)
- [ ] Implement sorting logic for players (activity-based or user-defined)
- [ ] Implement `MovePlayerToTop`
- [ ] Handle `playerctld` integration if present

### Formatter (`playerctl-formatter.c`)
- [ ] Implement template parsing logic (handling `{{ }}`)
- [ ] Implement tokenization for expressions
- [ ] Implement support for variables (`artist`, `title`, `status`, `playerName`, `playerInstance`, etc.)
- [ ] Implement helper functions:
    - [ ] `lc` (lowercase)
    - [ ] `uc` (uppercase)
    - [ ] `duration` (format duration)
    - [ ] `markup_escape`
    - [ ] `default`
    - [ ] `emoji`
    - [ ] `trunc`
- [ ] Implement math operations (`+`, `-`, `*`, `/`) for template variables

## CLI (`cmd/playerctl`) (`playerctl-cli.c`)
- [ ] Implement command-line argument parsing
- [ ] Implement main logic to handle commands (`play`, `pause`, `metadata`, etc.)
- [ ] Implement player selection logic (`--player`, `--ignore-player`, `--all-players`)
- [ ] Implement formatting output (`--format`)
- [ ] Implement follow mode (`--follow`) monitoring events and updating output

## Daemon (`cmd/playerctld`) (`playerctl-daemon.c`)
- [ ] Implement `playerctld` daemon structure
- [ ] Implement DBus service `org.mpris.MediaPlayer2.playerctld`
- [ ] Implement `Shift` and `Unshift` methods
- [ ] Implement `ActivePlayerChangeBegin` and `ActivePlayerChangeEnd` signals
- [ ] Maintain list of active players and their order
- [ ] Track player activity to update order
- [ ] Expose `PlayerNames` property

## Tests
- [ ] Port unit tests for Formatter
- [ ] Port unit tests for Player Manager
- [ ] Integration tests for DBus interaction

## Documentation
- [ ] Update README with Go installation instructions
- [ ] Write GoDocs for the library package
