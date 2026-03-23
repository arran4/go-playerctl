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
- `tui`
- `daemon`
- `loop [None|Track|Playlist]`
- `shuffle [On|Off|Toggle]`
- `volume [level]`
- `position [offset]`
- `open <uri>`
- `version`

# OPTIONS

- `--player`: comma-separated instance list.
- `--ignore-player`: comma-separated instance ignore list.
- `--all-players`: target all discovered players.
- `--list-all`: print discovered player names.
- `--format`: Go template format string.
- `--template-help`: print detailed help for format templates and exit.
- `--follow`: keep polling and print value changes.
- `--follow-interval`: polling interval (default: 1s).
- `--tui-scheme`: TUI control scheme (arrow, vim, winamp, emacs).
- `-v`, `--version`: print version string.

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

# FORMAT STRINGS

The Go port of `playerctl` uses standard Go `text/template` syntax for formatting output.

## Variables

The following variables are available in the template context:

- `.player`: The name of the player instance.
- `.status`: The playback status (e.g., Playing, Paused, Stopped).
- `.title`: The title of the current track.
- `.artist`: The artist of the current track.
- `.album`: The album of the current track.

## Functions

The following custom functions are available:

- `lc`: Lowercase string.
- `uc`: Uppercase string.
- `default <fallback> <value>`: Return fallback if value is empty.
- `duration <value>`: Format duration value.
- `markup_escape <value>`: HTML escape string.
- `emoji <status>`: Return an emoji for playback status.
- `trunc <max> <value>`: Truncate string to max length.
- `add <a> <b>`: Add two integers.
- `sub <a> <b>`: Subtract two integers.

# SEE ALSO

`README.md`, `cmd/goplayerctl/main.go`
