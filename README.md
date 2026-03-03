# go-playerctl

A Go port of Playerctl for controlling MPRIS-compatible media players over D-Bus.

> Status: **Go-complete baseline**. Core CLI, daemon, and library are implemented in Go. See docs in `docs/` for stability policy, acceptance checklist, and intentional deviations.

## Installation

You can download pre-compiled binaries for your platform from the [GitHub Releases page](https://github.com/arran4/go-playerctl/releases).

Alternatively, you can install from source using `go install`:

```bash
go install github.com/arran4/go-playerctl/cmd/goplayerctl@latest
```

## Quick start

```bash
go test ./...
go run ./cmd/goplayerctl --version
go run ./cmd/goplayerctl daemon --help
```

## CLI usage (`playerctl`)

```bash
go run ./cmd/goplayerctl [flags] <command>
```

### Supported flags

- `--player` comma-separated instance list (for example `vlc,spotify`)
- `--ignore-player` comma-separated instance ignore list
- `--all-players` run query/action for all discovered players
- `--list-all` print discovered player instances
- `--format` output format using Go template syntax
- `--follow` poll and print changes for query commands
- `--follow-interval` polling period for `--follow`
- `--version` print CLI version string

### Supported commands

- `play`
- `pause`
- `play-pause` / `playpause`
- `next`
- `previous`
- `status`
- `metadata`
- `daemon`

### Examples

```bash
# list players
go run ./cmd/goplayerctl --list-all

# query status for one player
go run ./cmd/goplayerctl --player vlc status

# query metadata for all players with formatted output
go run ./cmd/goplayerctl --all-players --format '{{ .player }}: {{ default .title "(none)" }}' metadata

# query metadata for a player showing artist, album, and title
go run ./cmd/playerctl --player spotify --format '{{ default .artist "Unknown Artist" }} - {{ default .album "Unknown Album" }} - {{ default .title "Unknown Title" }}' metadata

# follow status changes
go run ./cmd/goplayerctl --player spotify --follow status
```

## Daemon usage (`goplayerctl daemon`)

```bash
go run ./cmd/goplayerctl daemon [flags]
```

### Supported flags

- `--once` refresh and print discovered players once, then exit
- `--refresh-interval` refresh interval for daemon loop

### D-Bus service surface (current)

When not in `--once` mode, daemon attempts to export:

- Bus name: `org.mpris.MediaPlayer2.playerctld`
- Object path: `/org/mpris/MediaPlayer2`
- Interface: `com.github.altdesktop.playerctld`

Methods/properties currently exposed by the Go port:

- methods: `Shift`, `Unshift`
- signals emitted: `ActivePlayerChangeBegin`, `ActivePlayerChangeEnd`
- properties/accessors: `PlayerNames`, `ActivePlayer`

## Library usage (`pkg/playerctl`)

The package provides:

- enums and parsers (`PlaybackStatus`, `LoopStatus`, `Source`)
- typed errors (`ErrPlayerNotFound`, `InvalidCommandError`, `FormatError`)
- `Player` with MPRIS property getters/commands/metadata helpers
- `PlayerManager` for discovery/filtering/ordering helpers
- `Formatter` backed by Go `text/template`

### Minimal example

```go
package main

import (
    "fmt"
    "log"

    "github.com/arran4/go-playerctl/pkg/playerctl"
)

func main() {
    p, err := playerctl.NewPlayer("vlc", playerctl.SourceDBusSession)
    if err != nil {
        log.Fatal(err)
    }
    defer p.Close()

    status, err := p.PlaybackStatus()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(status)
}
```

## Formatting model

`Formatter` uses Go `text/template` and supports helper functions:

- `lc`, `uc`
- `default`
- `duration`
- `markup_escape`
- `emoji`
- `trunc`
- `add`, `sub`

Example:

```bash
go run ./cmd/goplayerctl --player spotify --format '{{ emoji .status }} {{ default .title "(none)" }}' status
```

## Documentation and references

- API stability policy: `docs/api_stability.md`
- Final acceptance checklist: `docs/final_acceptance_checklist.md`
- Intentional deviations: `docs/intentional_deviations.md`
- Legacy doc sources currently in tree:
  - `doc/playerctl.1.in`
  - `doc/reference/playerctl-docs.xml`
  - `doc/reference/version.xml.in`

## Development

```bash
go test ./...
go test -race ./...
```
