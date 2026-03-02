# Playerctl C → Go Port: Completion Checklist

This checklist tracks everything required to finish the port from the original C implementation to a fully working Go implementation.

## Current Snapshot (as of this update)
- The Go module and repository layout exist.
- Core enum and player-name types are implemented and tested.
- Most of the original C implementation is still present as commented source in Go files.
- `cmd/playerctl` and `cmd/playerctld` currently have stub `main()` functions.

---

## 0) Definition of Done
- [ ] `playerctl` CLI parity with the C version for core user workflows.
- [ ] `playerctld` daemon parity for active-player ordering and DBus interface behavior.
- [ ] Library API is stable, documented, and covered by unit/integration tests.
- [ ] Packaging/release pipeline produces installable artifacts.
- [ ] Legacy commented C blocks are removed once parity is reached.

---

## 1) Core Library: `pkg/playerctl`

### 1.1 Enums and shared types
- [x] Port playback/loop/source enums + parse/string conversion.
- [ ] Add any missing typed errors exported by the C API semantics (invalid command, formatting errors, player not found, etc.).
- [ ] Ensure all exported types have GoDoc comments and stable naming.

### 1.2 Player name handling
- [x] Port `PlayerName` struct and compare behavior.
- [ ] Validate compatibility of matching rules with C behavior (exact, wildcard, `%any`, instance matching).
- [ ] Add tests for edge cases from real MPRIS names.

### 1.3 Player implementation (`player.go`)
- [ ] Implement `Player` struct state + lifecycle (construct, close/cleanup).
- [ ] Implement constructors equivalent to C behavior:
  - [ ] `NewPlayer` (from instance/name)
  - [ ] `NewPlayerFromName`
- [ ] DBus wiring:
  - [ ] Connect to session/system bus based on source.
  - [ ] Subscribe to `PropertiesChanged`, `Seeked`, `NameOwnerChanged`.
  - [ ] Handle player disappearance / owner changes safely.
- [ ] Property getters:
  - [ ] `PlaybackStatus`
  - [ ] `LoopStatus`
  - [ ] `Shuffle`
  - [ ] `Volume`
  - [ ] `Position` (monotonic time adjustment semantics)
  - [ ] `Metadata`
  - [ ] `CanControl`, `CanPlay`, `CanPause`, `CanSeek`, `CanGoNext`, `CanGoPrevious`
- [ ] Commands/mutators:
  - [ ] `Play`, `Pause`, `PlayPause`, `Stop`
  - [ ] `Next`, `Previous`
  - [ ] `Seek`, `SetPosition`
  - [ ] `OpenUri`
  - [ ] `SetVolume`
  - [ ] `SetLoopStatus`
  - [ ] `SetShuffle`
- [ ] Metadata helpers:
  - [ ] `GetArtist`, `GetTitle`, `GetAlbum`, `GetTrackID`
  - [ ] Correctly parse array/string/object variants from DBus metadata.
- [ ] Add concurrency guards around signal callbacks + property cache updates.

### 1.4 Player manager (`player_manager.go`)
- [ ] Implement `PlayerManager` struct with synchronized player list.
- [ ] Player discovery via DBus `ListNames`.
- [ ] Dynamic add/remove via `NameOwnerChanged`.
- [ ] Selection/filtering rules (`--player`, ignore list, `%any`).
- [ ] Ordering behavior compatible with playerctld / activity sorting.
- [ ] `MovePlayerToTop` behavior parity.
- [ ] Bus-address and source handling parity with C behavior.

### 1.5 Formatter (`formatter.go`)
- [ ] Complete parser/tokenizer for `{{ ... }}` templates.
- [ ] Implement expression evaluation parity:
  - [ ] variables
  - [ ] function calls
  - [ ] string/number literals
  - [ ] arithmetic and unary ops
- [ ] Implement formatter helper functions parity:
  - [ ] `lc`
  - [ ] `uc`
  - [ ] `duration`
  - [ ] `markup_escape`
  - [ ] `default`
  - [ ] `emoji`
  - [ ] `trunc`
- [ ] Match null/missing-property behavior and error messages where practical.
- [ ] Add robust tests for parser errors and escaping behavior.

---

## 2) CLI Port: `cmd/playerctl`
- [ ] Replace stub `main()` with full CLI entrypoint.
- [ ] Port command-line flag parsing and validation.
- [ ] Implement command dispatch (`play`, `pause`, `status`, `metadata`, etc.).
- [ ] Implement player targeting:
  - [ ] `--player`
  - [ ] `--ignore-player`
  - [ ] `--all-players`
- [ ] Implement output modes:
  - [ ] default output
  - [ ] `--format`
  - [ ] metadata format handling
- [ ] Implement `--follow` mode:
  - [ ] subscribe to updates
  - [ ] suppress duplicate output lines
  - [ ] stable exit behavior on disconnect/errors
- [ ] Implement `--list-all`, `--version`, and command help parity.
- [ ] Match exit codes and stderr messaging behavior.

---

## 3) Daemon Port: `cmd/playerctld`
- [ ] Replace stub `main()` with working daemon lifecycle.
- [ ] Export DBus service `org.mpris.MediaPlayer2.playerctld`.
- [ ] Implement methods:
  - [ ] `Shift`
  - [ ] `Unshift`
- [ ] Implement signals:
  - [ ] `ActivePlayerChangeBegin`
  - [ ] `ActivePlayerChangeEnd`
- [ ] Implement properties:
  - [ ] `PlayerNames`
  - [ ] active-player tracking data
- [ ] Preserve ordering/activity semantics from C daemon.
- [ ] Handle race cases around player disappearance, seek buffering, and metadata churn.
- [ ] Implement robust shutdown and signal handling.

---

## 4) Test Parity and Validation

### 4.1 Unit tests (Go)
- [ ] Expand enum/name tests to include additional edge cases.
- [ ] Add formatter unit tests for functions, arithmetic, and parse errors.
- [ ] Add player-manager tests for ordering and filtering logic.
- [ ] Add player property/command tests with DBus mocks/fakes.

### 4.2 Integration tests
- [ ] Port/replace Python integration coverage for command behavior.
- [ ] Add end-to-end DBus tests for:
  - [ ] player discovery
  - [ ] follow mode
  - [ ] metadata formatting
  - [ ] daemon reordering

### 4.3 Regression matrix
- [ ] Build a feature parity matrix mapping C behavior → Go tests.
- [ ] Track unresolved intentional deviations with rationale.

---

## 5) Documentation
- [ ] Update `README.md` from “work in progress” to usage-focused docs.
- [ ] Document CLI commands/options with Go behavior examples.
- [ ] Document library API usage and stability expectations.
- [ ] Regenerate or port manpage/reference docs from legacy sources.
- [ ] Add migration notes for existing users of C-era packaging/behavior.

---

## 6) Tooling, CI, and Release
- [ ] Add CI workflow:
  - [ ] `go test ./...`
  - [ ] linting/static analysis
  - [ ] formatting checks
- [ ] Add race-detector and coverage jobs.
- [ ] Configure reproducible release process (GoReleaser or equivalent).
- [ ] Update distro/snap packaging artifacts to consume Go binaries.
- [ ] Verify daemon runtime assets are shipped/updated (`org.mpris.MediaPlayer2.playerctld` service files, DBus activation/service config, and startup scripts where applicable).
- [ ] Add changelog/release-note automation.

---

## 7) Cleanup Before Declaring Port Complete
- [ ] Remove large commented C source blocks from Go files.
- [ ] Remove obsolete C-era scripts/assets that are no longer used.
- [ ] Ensure repo tree reflects Go-native architecture.
- [ ] Run final parity acceptance checklist and tag first “Go-complete” release.

---

## Suggested Execution Order
1. Finish `pkg/playerctl` player + manager + formatter.
2. Implement working `cmd/playerctl` on top of finished library.
3. Implement `cmd/playerctld` and daemon-specific behavior.
4. Close parity gaps with integration tests.
5. Finalize docs, CI, packaging, and cleanup.
