# go-playerctl TODO

## Properties and Metadata
- [x] Implemented `LoopStatus` property
- [x] Implemented `Shuffle` property
- [x] Implemented `Position` property
- [ ] Implement `Rate` property getter/setter in `pkg/playerctl/player.go`
- [ ] Expose standard MPRIS and Xesam metadata keys (`mpris:trackid`, `mpris:artUrl`, `xesam:audioBPM`, etc.) to templates and output.
- [ ] Expose `position`, `volume`, `loopStatus`, `shuffle`, and `rate` variables to format templates.

## CLI Commands
- [ ] Implement `loop` command to get or set loop status.
- [ ] Implement `shuffle` command to get or set shuffle status.
- [ ] Implement `volume` command to get or set volume levels (absolute and relative).
- [ ] Implement `position` command to get or set position (absolute and relative).
- [ ] Implement `open` command to open URIs.

## TUI and Extended Features
- [ ] **TrackList Interface**: Expose TrackList methods (GetTracksMetadata, AddTrack, RemoveTrack, GoTo) and signals.
- [ ] **Playlists Interface**: Expose ActivatePlaylist, GetPlaylists, and signals.
- [ ] **TUI Enhancement**: Add a `p` for playlist mode that shows all tracks allowing navigation via full screen supporting pages, paging, and tables.
- [ ] **TUI Enhancement**: Support multiple playlists in TUI.
