% PLAYERCTL-GO(1)

# NAME

playerctl-go - control MPRIS media players from the Go port CLI

# SYNOPSIS

`goplayerctl [--version] [--list-all] [--all-players] [--player NAMES] [--ignore-player NAMES] [--format TEMPLATE] [--follow] [--follow-interval DURATION] COMMAND`

# DESCRIPTION

The Go port of `playerctl` controls media players implementing the MPRIS D-Bus interfaces.

# COMMANDS

- `play`
- `pause`
- `play-pause`
- `playpause`
- `next`
- `previous`
- `status`
- `metadata`

# OPTIONS

- `--player`: comma-separated instance list.
- `--ignore-player`: comma-separated instance ignore list.
- `--all-players`: target all discovered players.
- `--list-all`: print discovered player names.
- `--format`: Go template format string.
- `--follow`: keep polling and print value changes.
- `--follow-interval`: polling interval (default: 1s).
- `--version`: print version string.

# EXAMPLES

```bash
# list players
goplayerctl --list-all

# query status for one player
goplayerctl --player vlc status

# query metadata for all players with formatted output
goplayerctl --all-players --format '{{ .player }}: {{ default .title "(none)" }}' metadata

# query metadata for a player showing artist, album, and title
goplayerctl --player spotify --format '{{ default .artist "Unknown Artist" }} - {{ default .album "Unknown Album" }} - {{ default .title "Unknown Title" }}' metadata

# follow status changes
goplayerctl --player spotify --follow status
```

# FORMAT FUNCTIONS

- `lc`, `uc`
- `default`
- `duration`
- `markup_escape`
- `emoji`
- `trunc`
- `add`, `sub`

# SEE ALSO

`README.md`, `cmd/goplayerctl/main.go`
