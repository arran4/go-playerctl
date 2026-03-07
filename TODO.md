# go-playerctl TODO

## Properties and Metadata
- [x] Implemented `LoopStatus` property
- [x] Implemented `Shuffle` property
- [x] Implemented `Position` property
- [x] Implement `Rate` property getter/setter in `pkg/playerctl/player.go`
- [x] Expose standard MPRIS and Xesam metadata keys (`mpris:trackid`, `mpris:artUrl`, `xesam:audioBPM`, etc.) to templates and output.
- [x] Expose `position`, `volume`, `loopStatus`, `shuffle`, and `rate` variables to format templates.
- [x] Expose tracklist and playlist information (e.g., active playlist name, track counts) as format template variables.
- [x] Add helper template functions for interacting with and displaying tracklists and playlists clearly.
- [x] Add robust examples to README, help menus, and documentation showing how to effectively output tracklist and playlist data.

## CLI Commands
- [x] Implement `loop` command to get or set loop status.
- [x] Implement `shuffle` command to get or set shuffle status.
- [x] Implement `volume` command to get or set volume levels (absolute and relative).
- [x] Implement `position` command to get or set position (absolute and relative).
- [x] Implement `open` command to open URIs.
- [x] Implement `playlist` command to list, switch, or show details of available playlists.
- [x] Implement `tracklist` command to list, switch, or show details of tracks in the current tracklist.

## TUI and Extended Features
- [x] **TrackList Interface**: Expose TrackList methods (GetTracksMetadata, AddTrack, RemoveTrack, GoTo) and signals to the underlying D-Bus API.
- [x] **Playlists Interface**: Expose ActivatePlaylist, GetPlaylists, and signals to the underlying D-Bus API.
- [x] **TUI Enhancement**: Add a `p` for playlist mode that shows all tracks allowing navigation via full screen supporting pages, paging, and tables.
- [x] **TUI Enhancement**: Add a `t` for tracklist mode that shows the current playing tracklist allowing navigation via full screen supporting pages, paging, and tables.
- [x] **TUI Enhancement**: Support multiple playlists and tracklist in TUI.
