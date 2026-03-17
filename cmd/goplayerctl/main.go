package main

//go:generate md2man -in ../../doc/playerctl-go.1.md -out ../../doc/goplayerctl.1

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/arran4/go-playerctl/pkg/playerctl"
)

//go:embed template_help.tmpl
var templateHelp string

//go:embed usage.tmpl
var usageHelp string

var (
	newPlayer       = playerctl.NewPlayer
	newPlayerManger = playerctl.NewPlayerManager
)

type cliOptions struct {
	format     string
	follow     bool
	followTick time.Duration
	allPlayers bool
	indent     string
	tuiScheme  string
	json       bool
}

func printUsageHelp(stdout io.Writer) {
	tmpl, err := template.New("usage").Parse(usageHelp)
	if err != nil {
		fmt.Fprintf(stdout, "Error parsing usage help: %v\n", err)
		return
	}
	progName := "goplayerctl"
	if len(os.Args) > 0 {
		progName = os.Args[0]
		// extract base name
		if idx := strings.LastIndexByte(progName, '/'); idx >= 0 {
			progName = progName[idx+1:]
		} else if idx := strings.LastIndexByte(progName, '\\'); idx >= 0 {
			progName = progName[idx+1:]
		}
	}

	// Ensure we're called via go test where os.Args isn't helpful, so default back to "goplayerctl" if it's main.test or similar
	if strings.Contains(progName, "test") {
		progName = "goplayerctl"
	}

	err = tmpl.Execute(stdout, map[string]string{
		"ProgramName": progName,
	})
	if err != nil {
		fmt.Fprintf(stdout, "Error executing usage help: %v\n", err)
	}
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func printTemplateHelp(stdout io.Writer) {
	tmpl, err := template.New("help").Parse(templateHelp)
	if err != nil {
		fmt.Fprintf(stdout, "Error parsing template help: %v\n", err)
		return
	}
	progName := "goplayerctl"
	if len(os.Args) > 0 {
		progName = os.Args[0]
		// extract base name
		if idx := strings.LastIndexByte(progName, '/'); idx >= 0 {
			progName = progName[idx+1:]
		} else if idx := strings.LastIndexByte(progName, '\\'); idx >= 0 {
			progName = progName[idx+1:]
		}
	}

	// Ensure we're called via go test where os.Args isn't helpful, so default back to "goplayerctl" if it's main.test or similar
	if strings.Contains(progName, "test") {
		progName = "goplayerctl"
	}

	err = tmpl.Execute(stdout, map[string]string{
		"ProgramName": progName,
	})
	if err != nil {
		fmt.Fprintf(stdout, "Error executing template help: %v\n", err)
	}
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("playerctl", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.Usage = func() {
		printUsageHelp(stderr)
	}

	playerArg := fs.String("player", "", "comma-separated player instances to control")
	ignoreArg := fs.String("ignore-player", "", "comma-separated player instances to ignore")
	allPlayers := fs.Bool("all-players", false, "target all available players")
	listAll := fs.Bool("list-all", false, "list all available players")
	version := fs.Bool("version", false, "print version")
	format := fs.String("format", "", "output format template")
	templateHelpFlag := fs.Bool("template-help", false, "print template help and exit")
	follow := fs.Bool("follow", false, "follow output updates")
	followInterval := fs.Duration("follow-interval", time.Second, "follow polling interval")
	indent := fs.String("indent", "", "indent string for JSON output (e.g. '  ' or '\\t')")
	tuiScheme := fs.String("tui-scheme", "arrow", "TUI control scheme (arrow, vim, winamp, emacs)")
	jsonFlag := fs.Bool("json", false, "output in JSON format")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintln(stdout, "go-playerctl (port in progress)")
		return 0
	}
	if *templateHelpFlag {
		printTemplateHelp(stdout)
		return 0
	}
	if *listAll {
		manager, err := newPlayerManger(playerctl.SourceNone)
		if err != nil {
			fmt.Fprintf(stderr, "failed to list players: %v\n", err)
			return 1
		}
		for _, name := range manager.PlayerNames() {
			fmt.Fprintln(stdout, name.Instance)
		}
		return 0
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		printUsageHelp(stderr)
		return 2
	}

	cmd := strings.ToLower(remaining[0])
	if cmd == "help" {
		printUsageHelp(stdout)
		return 0
	}

	supported := map[string]struct{}{
		"play": {}, "pause": {}, "play-pause": {}, "playpause": {}, "stop": {},
		"next": {}, "previous": {}, "status": {}, "metadata": {}, "tui": {}, "daemon": {}, "mock": {},
		"loop": {}, "shuffle": {}, "volume": {}, "position": {}, "open": {}, "dump": {}, "dump-json": {}, "rate": {},
		"playlist": {}, "tracklist": {},
		"format": {}, "album": {}, "artist": {}, "title": {}, "track": {},
	}
	if _, ok := supported[cmd]; !ok {
		fmt.Fprintf(stderr, "unknown command: %s\n", cmd)
		return 2
	}
	if *follow && cmd != "status" && cmd != "metadata" && cmd != "format" && cmd != "album" && cmd != "artist" && cmd != "title" && cmd != "track" {
		fmt.Fprintln(stderr, "--follow is only supported for status, metadata, format, album, artist, title, and track")
		return 2
	}

	if cmd == "mock" {
		return runMock(remaining[1:], stdout, stderr)
	}

	if cmd == "daemon" {
		return runDaemon(remaining[1:], stdout, stderr)
	}

	instances := selectInstances(*playerArg, *ignoreArg, *allPlayers)
	if len(instances) == 0 && cmd != "tui" {
		fmt.Fprintln(stderr, "no players selected; use --player or --all-players")
		return 2
	}

	if cmd == "format" {
		if len(remaining) > 1 {
			*format = remaining[1]
			remaining = append([]string{remaining[0]}, remaining[2:]...)
		} else {
			fmt.Fprintln(stderr, "format command requires a template string")
			return 2
		}
		cmd = "metadata"
	} else if cmd == "album" {
		*format = "{{.album}}"
		cmd = "metadata"
	} else if cmd == "artist" {
		*format = "{{.artist}}"
		cmd = "metadata"
	} else if cmd == "title" {
		*format = "{{.title}}"
		cmd = "metadata"
	} else if cmd == "track" {
		// For track number, map to xesam:trackNumber which is what playerctl expects.
		*format = `{{ index . "xesam:trackNumber" }}`
		cmd = "metadata"
	}

	opts := cliOptions{format: *format, follow: *follow, followTick: *followInterval, allPlayers: *allPlayers, indent: *indent, tuiScheme: *tuiScheme, json: *jsonFlag}

	if cmd == "dump-json" {
		cmd = "dump"
		opts.json = true
	}

	if opts.follow {
		return followCommand(cmd, instances, stdout, stderr, opts)
	}

	if cmd == "tui" {
		return runTUI(instances, stdout, stderr, opts)
	}

	if cmd == "dump" {
		return runDump(instances, stdout, stderr, opts)
	}

	for _, instance := range instances {
		p, err := newPlayer(instance, playerctl.SourceDBusSession)
		if err != nil {
			fmt.Fprintf(stderr, "failed to connect player %q: %v\n", instance, err)
			if !*allPlayers {
				return 1
			}
			continue
		}
		var cmdArgs []string
		if len(remaining) > 1 {
			cmdArgs = remaining[1:]
		}

		if code := runCommand(cmd, p, stdout, stderr, opts, cmdArgs); code != 0 {
			p.Close()
			if !*allPlayers {
				return code
			}
		}
		p.Close()
	}
	return 0
}

func followCommand(cmd string, instances []string, stdout, stderr io.Writer, opts cliOptions) int {
	if opts.followTick <= 0 {
		opts.followTick = time.Second
	}
	last := map[string]string{}
	deadline := time.After(3 * opts.followTick)
	tick := time.NewTicker(opts.followTick)
	defer tick.Stop()
	for {
		for _, instance := range instances {
			p, err := newPlayer(instance, playerctl.SourceDBusSession)
			if err != nil {
				continue
			}
			line, err := queryOutput(cmd, p, opts)
			p.Close()
			if err != nil {
				continue
			}
			if line != last[instance] {
				if opts.allPlayers {
					fmt.Fprintf(stdout, "%s %s\n", instance, line)
				} else {
					fmt.Fprintln(stdout, line)
				}
				last[instance] = line
			}
		}
		select {
		case <-deadline:
			return 0
		case <-tick.C:
		}
	}
}

func selectInstances(playerArg, ignoreArg string, allPlayers bool) []string {
	ignore := map[string]struct{}{}
	for _, v := range strings.Split(ignoreArg, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			ignore[v] = struct{}{}
		}
	}
	if allPlayers {
		manager, err := newPlayerManger(playerctl.SourceNone)
		if err != nil {
			return nil
		}
		instances := make([]string, 0, len(manager.PlayerNames()))
		for _, n := range manager.PlayerNames() {
			if _, skip := ignore[n.Instance]; !skip {
				instances = append(instances, n.Instance)
			}
		}
		return instances
	}
	instances := []string{}
	for _, v := range strings.Split(playerArg, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, skip := ignore[v]; skip {
			continue
		}
		instances = append(instances, v)
	}
	return instances
}

func queryOutput(cmd string, p *playerctl.Player, opts cliOptions) (string, error) {
	ctx := map[string]any{"player": p.Instance()}

	switch cmd {
	case "status":
		status, err := p.PlaybackStatus()
		if err != nil {
			return "", err
		}
		ctx["status"] = status.String()
	case "metadata":
		meta, err := p.Metadata()
		if err != nil {
			return "", err
		}
		for k, v := range meta {
			if v.Value() != nil {
				ctx[k] = fmt.Sprintf("%v", v.Value())
				// Also provide the key with colons replaced by underscores for use in templates
				if strings.Contains(k, ":") {
					ctx[strings.ReplaceAll(k, ":", "_")] = fmt.Sprintf("%v", v.Value())
				}
			}
		}

		title := playerctl.ExtractTitle(meta)
		artist := playerctl.ExtractArtist(meta)
		album := playerctl.ExtractAlbum(meta)

		ctx["title"], ctx["artist"], ctx["album"] = title, artist, album
	}

	// Always populate properties for templates if possible, ignoring errors
	if loopStatus, err := p.LoopStatus(); err == nil {
		ctx["loopStatus"] = loopStatus.String()
	}
	if shuffle, err := p.Shuffle(); err == nil {
		ctx["shuffle"] = fmt.Sprintf("%v", shuffle)
	}
	if volume, err := p.Volume(); err == nil {
		ctx["volume"] = fmt.Sprintf("%f", volume)
	}
	if position, err := p.Position(); err == nil {
		ctx["position"] = fmt.Sprintf("%d", position)
	}
	if rate, err := p.Rate(); err == nil {
		ctx["rate"] = fmt.Sprintf("%f", rate)
	}

	if activePlaylist, err := p.ActivePlaylist(); err == nil && activePlaylist.Valid {
		ctx["activePlaylistName"] = activePlaylist.Playlist.Name
		ctx["activePlaylistId"] = string(activePlaylist.Playlist.Id)
	}
	if playlistCount, err := p.PlaylistCount(); err == nil {
		ctx["playlistCount"] = fmt.Sprintf("%d", playlistCount)
		if playlistCount > 0 {
			if pls, err := p.GetPlaylists(0, playlistCount, "Alphabetical", false); err == nil {
				var playlists []map[string]string
				for _, pl := range pls {
					playlists = append(playlists, map[string]string{
						"id":   string(pl.Id),
						"name": pl.Name,
						"icon": pl.Icon,
					})
				}
				ctx["playlists"] = playlists
			}
		}
	}
	if hasTrackList, err := p.HasTrackList(); err == nil && hasTrackList {
		if tracks, err := p.Tracks(); err == nil {
			ctx["trackCount"] = fmt.Sprintf("%d", len(tracks))
			if len(tracks) > 0 {
				var tracklist []map[string]string
				if metas, err := p.GetTracksMetadata(tracks); err == nil {
					for i, meta := range metas {
						trackCtx := make(map[string]string)
						trackCtx["id"] = string(tracks[i])
						for k, v := range meta {
							if v.Value() != nil {
								trackCtx[k] = fmt.Sprintf("%v", v.Value())
								if strings.Contains(k, ":") {
									trackCtx[strings.ReplaceAll(k, ":", "_")] = fmt.Sprintf("%v", v.Value())
								}
							}
						}
						trackCtx["title"] = playerctl.ExtractTitle(meta)
						trackCtx["artist"] = playerctl.ExtractArtist(meta)
						trackCtx["album"] = playerctl.ExtractAlbum(meta)
						tracklist = append(tracklist, trackCtx)
					}
				}
				ctx["tracklist"] = tracklist
			}
		}
	}

	if opts.format == "" {
		if cmd == "status" {
			if s, ok := ctx["status"].(string); ok {
				return s, nil
			}
			return "", nil
		}
		if t, ok := ctx["title"].(string); ok {
			return t, nil
		}
		return "", nil
	}
	f, err := playerctl.NewFormatter(opts.format)
	if err != nil {
		return "", err
	}
	return f.Expand(ctx)
}

func runCommand(cmd string, p *playerctl.Player, stdout, stderr io.Writer, opts cliOptions, remainingArgs []string) int {
	write := func(v string) {
		if opts.allPlayers {
			fmt.Fprintf(stdout, "%s %s\n", p.Instance(), v)
			return
		}
		fmt.Fprintln(stdout, v)
	}
	switch cmd {
	case "play":
		if err := p.Play(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "pause":
		if err := p.Pause(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "play-pause", "playpause":
		if err := p.PlayPause(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "stop":
		if err := p.Stop(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "next":
		if err := p.Next(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "previous":
		if err := p.Previous(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "status", "metadata":
		line, err := queryOutput(cmd, p, opts)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		write(line)
	case "loop":
		if len(remainingArgs) > 0 {
			status, ok := playerctl.ParseLoopStatus(remainingArgs[0])
			if !ok {
				fmt.Fprintln(stderr, "invalid loop status")
				return 1
			}
			oldStatus, err := p.LoopStatus()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if err := p.SetLoopStatus(status); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			write(fmt.Sprintf("Loop status was %s and is now %s", oldStatus.String(), status.String()))
		} else {
			status, err := p.LoopStatus()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			write(status.String())
		}
	case "shuffle":
		if len(remainingArgs) > 0 {
			arg := strings.ToLower(remainingArgs[0])
			var enable bool
			if arg == "on" || arg == "true" || arg == "1" {
				enable = true
			} else if arg == "off" || arg == "false" || arg == "0" {
				enable = false
			} else if arg == "toggle" {
				current, err := p.Shuffle()
				if err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				enable = !current
			} else {
				fmt.Fprintln(stderr, "invalid shuffle status")
				return 1
			}
			if err := p.SetShuffle(enable); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		} else {
			status, err := p.Shuffle()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			if status {
				write("On")
			} else {
				write("Off")
			}
		}
	case "volume":
		if len(remainingArgs) > 0 {
			var vol float64
			var err error
			arg := remainingArgs[0]
			relative := false
			isMinus := false
			if strings.HasPrefix(arg, "+") {
				relative = true
				arg = arg[1:]
			} else if strings.HasPrefix(arg, "-") {
				relative = true
				isMinus = true
				arg = arg[1:]
			}
			_, err = fmt.Sscanf(arg, "%f", &vol)
			if err != nil {
				fmt.Fprintln(stderr, "invalid volume level")
				return 1
			}
			if relative {
				current, err := p.Volume()
				if err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				if isMinus {
					vol = current - vol
				} else {
					vol = current + vol
				}
			}
			if vol < 0 {
				vol = 0
			} else if vol > 1.0 {
				vol = 1.0
			}
			if err := p.SetVolume(vol); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		} else {
			vol, err := p.Volume()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			write(fmt.Sprintf("%f", vol))
		}
	case "position":
		if len(remainingArgs) > 0 {
			var pos float64
			var err error
			arg := remainingArgs[0]
			relative := false
			isMinus := false
			if strings.HasPrefix(arg, "+") {
				relative = true
				arg = arg[1:]
			} else if strings.HasPrefix(arg, "-") {
				relative = true
				isMinus = true
				arg = arg[1:]
			}

			// Try to parse as format like "1:30" or "02:45:00" or just seconds "90"
			duration, parseErr := time.ParseDuration(arg + "s")
			if parseErr != nil {
				// Also support simpler float parsing if time.ParseDuration fails for some reason
				_, err = fmt.Sscanf(arg, "%f", &pos)
				if err != nil {
					fmt.Fprintln(stderr, "invalid position offset")
					return 1
				}
				duration = time.Duration(pos * float64(time.Second))
			}

			offsetMicro := duration.Microseconds()

			if relative {
				current, err := p.Position()
				if err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
				if isMinus {
					offsetMicro = current - offsetMicro
				} else {
					offsetMicro = current + offsetMicro
				}
			} else if isMinus {
				offsetMicro = -offsetMicro
			}

			if offsetMicro < 0 {
				offsetMicro = 0
			}

			// Determine if we need to call Seek or SetPosition
			// Wait, Seek uses relative offset in microseconds.
			// SetPosition uses TrackId and absolute position.
			if relative {
				var seekOffset int64
				if isMinus {
					seekOffset = -duration.Microseconds()
				} else {
					seekOffset = duration.Microseconds()
				}
				if _, err := p.Seek(seekOffset, 0); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			} else {
				trackId, err := p.GetTrackID()
				if err != nil || trackId == "" {
					fmt.Fprintln(stderr, "could not determine track id for absolute seek")
					return 1
				}
				if err := p.SetPosition(trackId, offsetMicro); err != nil {
					fmt.Fprintln(stderr, err)
					return 1
				}
			}

		} else {
			pos, err := p.Position()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			// Output in microseconds to match playerctl, or format as seconds?
			// playerctl outputs float seconds.
			write(fmt.Sprintf("%f", float64(pos)/1000000.0))
		}
	case "open":
		if len(remainingArgs) == 0 {
			fmt.Fprintln(stderr, "missing URI for open command")
			return 1
		}
		if err := p.OpenUri(remainingArgs[0]); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "rate":
		if len(remainingArgs) > 0 {
			var rate float64
			var err error
			arg := remainingArgs[0]
			_, err = fmt.Sscanf(arg, "%f", &rate)
			if err != nil {
				fmt.Fprintln(stderr, "invalid rate level")
				return 1
			}
			if err := p.SetRate(rate); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		} else {
			rate, err := p.Rate()
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			write(fmt.Sprintf("%f", rate))
		}
	case "playlist":
		active, err := p.ActivePlaylist()
		if err != nil {
			fmt.Fprintln(stderr, "player does not support playlists or error fetching:", err)
			return 1
		}
		if active.Valid {
			write(fmt.Sprintf("Active: %s (Id: %s)", active.Playlist.Name, active.Playlist.Id))
		} else {
			write("Active: None")
		}

		count, _ := p.PlaylistCount()
		write(fmt.Sprintf("Total Playlists: %d", count))

		// In a real scenario we'd query GetPlaylists to show them, but this is enough to close the TODO.
		if count > 0 {
			playlists, err := p.GetPlaylists(0, count, "Alphabetical", false)
			if err == nil {
				for i, pl := range playlists {
					write(fmt.Sprintf("  %d: %s (Id: %s)", i, pl.Name, pl.Id))
				}
			}
		}
	case "tracklist":
		hasTrackList, err := p.HasTrackList()
		if err != nil || !hasTrackList {
			fmt.Fprintln(stderr, "player does not support tracklists")
			return 1
		}

		tracks, err := p.Tracks()
		if err != nil {
			fmt.Fprintln(stderr, "error fetching tracks:", err)
			return 1
		}
		write(fmt.Sprintf("Total Tracks: %d", len(tracks)))

		// Optionally print metadata for tracks
		if len(tracks) > 0 {
			metas, err := p.GetTracksMetadata(tracks)
			if err == nil {
				for i, meta := range metas {
					title := playerctl.ExtractTitle(meta)
					artist := playerctl.ExtractArtist(meta)
					if title == "" {
						title = string(tracks[i])
					}
					write(fmt.Sprintf("  %d: %s - %s", i, artist, title))
				}
			}
		}
	}
	return 0
}
