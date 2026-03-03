# Go Port Parity Matrix (C behavior vs current Go behavior)

This matrix tracks high-value parity outcomes and links each item to current tests.

## `playerctl` CLI edge-case parity

| Scenario | Expected behavior | Current Go status | Test reference |
|---|---|---|---|
| `--version` | exit `0`, version text on stdout | ✅ | `cmd/playerctl/main_test.go::TestRunValidationAndVersion` |
| missing command | exit `2`, usage-style stderr | ✅ (`missing command`) | `integration_cli_test.go::TestPlayerctlMissingCommandIntegration` |
| unknown command | exit `2`, stderr contains command name | ✅ | `cmd/playerctl/main_test.go::TestRunValidationAndVersion` |
| connection failure | exit `1`, stderr indicates connection failure | ✅ | `cmd/playerctl/main_test.go::TestRunConnectionFailure` |
| invalid `--follow` command | exit `2`, stderr validation message | ✅ | `cmd/playerctl/main_test.go::TestRunFollowValidation` |

## `playerctld` ordering/race-case validation

| Scenario | Expected behavior | Current Go status | Test reference |
|---|---|---|---|
| `Shift` rotates order | first player moves to end | ✅ | `cmd/playerctld/main_test.go::TestDaemonShiftUnshift` |
| `Unshift` rotates reverse | last player moves to front | ✅ | `cmd/playerctld/main_test.go::TestDaemonShiftUnshift` |
| active player updates with reordering | `ActivePlayer` tracks top player and timestamp | ✅ | `cmd/playerctld/main_test.go::TestDaemonActivePlayerTracking` |
| concurrent shift/unshift safety | no panic/race, stable list size | ✅ | `cmd/playerctld/main_test.go::TestDaemonConcurrentShiftUnshift` |

## Remaining parity gaps

- Full command output and stderr text parity against C implementation for all commands.
- End-to-end DBus integration parity for discovery/follow/metadata formatting/daemon reordering.
