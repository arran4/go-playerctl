% PLAYERCTL-GO(1)

# NAME

playerctl-go - control MPRIS media players from the Go port CLI

# SYNOPSIS

`go run ./cmd/playerctl [--version] [--list-all] [--all-players] [--player NAMES] [--ignore-player NAMES] [--format TEMPLATE] [--follow] [--follow-interval DURATION] COMMAND`

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

# FORMAT FUNCTIONS

- `lc`, `uc`
- `default`
- `duration`
- `markup_escape`
- `emoji`
- `trunc`
- `add`, `sub`

# SEE ALSO

`README.md`, `docs/final_acceptance_checklist.md`, `cmd/playerctl/main.go`
